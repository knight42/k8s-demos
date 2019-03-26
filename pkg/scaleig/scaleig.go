package scaleig

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/knight42/k8s-tools/pkg/utils"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/kubernetes"
)

type ScaleInstanceGroupOptions struct {
	igName  string
	cluster string
	region  string
	size    int

	configFlags *genericclioptions.ConfigFlags
	clientSet   *kubernetes.Clientset
	awsSession  *session.Session
}

func skipPod(pod *corev1.Pod) bool {
	// skip pods created by DaemonSet
	ctrler := metav1.GetControllerOf(pod)
	if ctrler != nil && ctrler.Kind == "DaemonSet" {
		return true
	}

	// skip pods which have local storage
	for _, volume := range pod.Spec.Volumes {
		if volume.EmptyDir != nil {
			return true
		}
	}

	// skip mirror pods
	if _, ok := pod.Annotations[corev1.MirrorPodAnnotationKey]; ok {
		return true
	}

	return false
}

func NewScaleInstanceGroupOptions() *ScaleInstanceGroupOptions {
	return &ScaleInstanceGroupOptions{
		configFlags: genericclioptions.NewConfigFlags(true),
	}
}

func NewCmd() *cobra.Command {
	o := NewScaleInstanceGroupOptions()

	cmd := &cobra.Command{
		Use:  "kubectl scaleig [NAME] [flags]",
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			utils.CheckError(o.Complete(cmd, args))
			utils.CheckError(o.Validate())
			utils.CheckError(o.Run())
		},
	}
	flags := cmd.Flags()
	flags.IntVar(&o.size, "size", -1, "Desired size of instance group")
	flags.StringVarP(&o.cluster, "cluster-name", "c", os.Getenv("KOPS_CLUSTER_NAME"), "Name of kops cluster. Overrides KOPS_CLUSTER_NAME environment variable")
	flags.StringVar(&o.region, "region", "cn-north-1", "AWS region")

	o.configFlags.AddFlags(cmd.PersistentFlags())
	return cmd
}

func (o *ScaleInstanceGroupOptions) Complete(cmd *cobra.Command, args []string) error {
	var (
		err error
	)

	restCfg, err := o.configFlags.ToRESTConfig()
	if err != nil {
		return err
	}

	o.clientSet, err = kubernetes.NewForConfig(restCfg)
	if err != nil {
		return err
	}

	o.awsSession, err = session.NewSession(&aws.Config{Region: aws.String(o.region)})
	if err != nil {
		return err
	}

	if len(args) < 1 {
		return fmt.Errorf("missing instance group")
	}

	o.igName = args[0]

	return nil
}

func (o *ScaleInstanceGroupOptions) Validate() error {
	if len(o.igName) == 0 {
		return fmt.Errorf("missing instance group")
	}

	if len(o.cluster) == 0 {
		return fmt.Errorf("missing cluster name")
	}

	if o.size == -1 {
		return fmt.Errorf("must specify the desired size of the given instance group: --size")
	}

	if o.size < 0 {
		return fmt.Errorf("invalid size: `%d`", o.size)
	}
	return nil
}

func (o *ScaleInstanceGroupOptions) ensureSchedulability(nodeNames []string, schedulable bool) error {
	nodesAPI := o.clientSet.CoreV1().Nodes()
	patchBytes := []byte(fmt.Sprintf("{\"spec\":{\"unschedulable\":%v}}", !schedulable))

	for _, name := range nodeNames {
		_, err := nodesAPI.Patch(name, types.StrategicMergePatchType, patchBytes)
		if err != nil {
			return err
		}

		if schedulable {
			fmt.Printf("node %s uncordoned\n", name)
		} else {
			fmt.Printf("node %s cordoned\n", name)
		}
	}
	return nil
}

func (o *ScaleInstanceGroupOptions) evictPodsOnNode(nodeName string) error {
	fieldSelector := fields.OneTermEqualSelector("spec.nodeName", nodeName)

	corev1API := o.clientSet.CoreV1()

	lstOpt := metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	}
	podList, err := corev1API.Pods(metav1.NamespaceAll).List(lstOpt)
	if err != nil {
		return fmt.Errorf("list pods: %s", err)
	}

	policy := metav1.DeletePropagationForeground
	for _, pod := range podList.Items {
		if skipPod(&pod) {
			fmt.Printf("Ignoring pod %s/%s\n", pod.Namespace, pod.Name)
			continue
		}

		eviction := &policyv1beta1.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name: pod.Name,
			},
			DeleteOptions: &metav1.DeleteOptions{
				PropagationPolicy: &policy,
			},
		}
		err = corev1API.Pods(pod.Namespace).Evict(eviction)
		if err != nil {
			return fmt.Errorf("evict pod: %s", err)
		}

		fmt.Printf("pod %s/%s evicted\n", pod.Namespace, pod.Name)
	}

	fmt.Printf("node %s evicted\n", nodeName)
	return nil
}

func (o *ScaleInstanceGroupOptions) Run() error {
	autoscalingSvc := autoscaling.New(o.awsSession)

	asgName := fmt.Sprintf("%s.%s", o.igName, o.cluster)
	asgNamePtr := aws.String(asgName)

	descASGInput := &autoscaling.DescribeAutoScalingGroupsInput{
		AutoScalingGroupNames: []*string{aws.String(asgName)},
	}

	descASGOutput, err := autoscalingSvc.DescribeAutoScalingGroups(descASGInput)
	if err != nil {
		return err
	}

	if len(descASGOutput.AutoScalingGroups) == 0 {
		return fmt.Errorf("not found: auto scaling group `%s`", asgName)
	} else if len(descASGOutput.AutoScalingGroups) > 1 {
		return fmt.Errorf("found more than one auto scaling group: `%s`", asgName)
	}

	asg := descASGOutput.AutoScalingGroups[0]
	if len(asg.Instances) == 0 {
		return fmt.Errorf("not found: auto scaling group `%s` contains no instance", asgName)
	}

	instanceCount := len(asg.Instances)
	if o.size == instanceCount {
		fmt.Println("No changes")
		return nil
	} else if o.size > instanceCount {
		fmt.Printf("Current: %d, Desired: %d\n", instanceCount, o.size)
		fmt.Printf("Use `kops edit ig %s` instead\n", asgName)
		return nil
	}

	delta := instanceCount - o.size
	instanceIDs := make([]*string, delta)
	for i, inst := range asg.Instances[:delta] {
		instanceIDs[i] = inst.InstanceId
	}

	ec2Svc := ec2.New(o.awsSession)
	descInstInput := &ec2.DescribeInstancesInput{
		InstanceIds: instanceIDs,
	}
	descInstOutput, err := ec2Svc.DescribeInstances(descInstInput)
	if err != nil {
		return err
	}

	nodeNames := make([]string, delta)

	i := 0
	for _, reservation := range descInstOutput.Reservations {
		for _, inst := range reservation.Instances {
			nodeNames[i] = aws.StringValue(inst.PrivateDnsName)
			i += 1
		}
	}

	err = o.ensureSchedulability(nodeNames, false)
	if err != nil {
		return err
	}

	for _, nodeName := range nodeNames {
		fmt.Printf("Evicting pods on node %s\n", nodeName)
		err := o.evictPodsOnNode(nodeName)
		if err != nil {
			return err
		}
	}

	desiredSize := aws.Int64(aws.Int64Value(asg.MinSize) - int64(delta))
	updateASGInput := autoscaling.UpdateAutoScalingGroupInput{
		MinSize:              desiredSize,
		MaxSize:              desiredSize,
		AutoScalingGroupName: asgNamePtr,
	}
	fmt.Printf("Updating auto scaling group `%s`\n", asgName)
	_, err = autoscalingSvc.UpdateAutoScalingGroup(&updateASGInput)
	if err != nil {
		return fmt.Errorf("update auto scaling group: %s", err)
	}

	shouldDecrementDesiredCapacity := true
	detachInput := autoscaling.DetachInstancesInput{
		InstanceIds:                    instanceIDs,
		AutoScalingGroupName:           asgNamePtr,
		ShouldDecrementDesiredCapacity: &shouldDecrementDesiredCapacity,
	}
	fmt.Printf("Detaching instances from auto scaling group `%s`\n", asgName)
	_, err = autoscalingSvc.DetachInstances(&detachInput)
	if err != nil {
		return fmt.Errorf("detach instances: %s", err)
	}

	terminateInstInput := ec2.TerminateInstancesInput{
		InstanceIds: instanceIDs,
	}
	fmt.Printf("Terminating instances: %v\n", aws.StringValueSlice(instanceIDs))
	_, err = ec2Svc.TerminateInstances(&terminateInstInput)
	if err != nil {
		return fmt.Errorf("terminate instances: %s", err)
	}

	fmt.Printf("Please run `kops edit ig %s` to update kops state\n", o.igName)
	return nil
}
