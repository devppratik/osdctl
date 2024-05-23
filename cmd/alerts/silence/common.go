package silence

import (
	"context"
	"fmt"

	"github.com/openshift/osdctl/cmd/cluster"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"k8s.io/kubectl/pkg/scheme"
)

// Labels represents a set of labels associated with an alert.
type AlertLabels struct {
	Alertname string `json:"alertname"`
	Severity  string `json:"severity"`
}

// Status represents a set of state associated with an alert.
type AlertStatus struct {
	State string `json:"state"`
}

// Annotations represents a set of summary/description associated with an alert.
type AlertAnnotations struct {
	Summary string `json:"summary"`
}

// Alert represents a set of above declared struct Labels,Status and annoataions
type Alert struct {
	Labels      AlertLabels      `json:"labels"`
	Status      AlertStatus      `json:"status"`
	Annotations AlertAnnotations `json:"annotations"`
}

type SilenceID struct {
	ID string `json:"id"`
}

type SilenceMatchers struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type SilenceStatus struct {
	State string `json:"state"`
}

type Silence struct {
	ID        string            `json:"id"`
	Matchers  []SilenceMatchers `json:"matchers"`
	Status    SilenceStatus     `json:"status"`
	Comment   string            `json:"comment"`
	CreatedBy string            `json:"createdBy"`
	EndsAt    string            `json:"endsAt"`
	StartsAt  string            `json:"startsAt"`
}

const (
	AccountNamespace = "openshift-monitoring"
	ContainerName    = "alertmanager"
	LocalHostUrl     = "http://localhost:9093"
	PrimaryPod       = "alertmanager-main-0"
	SecondaryPod     = "alertmanager-main-1"
)

// ExecInPod is designed to execute a command inside a Kubernetes pod and capture its output.
func ExecInPod(kubeconfig *rest.Config, clientset *kubernetes.Clientset, cmd []string) (string, error) {
	var cmdOutput string
	var err error

	// Attempt to execute with the primary pod
	cmdOutput, err = ExecWithPod(kubeconfig, clientset, PrimaryPod, cmd)
	if err == nil {
		return cmdOutput, nil // Successfully executed
	}

	// If execution with primary pod fails, try with the secondary pod
	cmdOutput, err = ExecWithPod(kubeconfig, clientset, SecondaryPod, cmd)
	if err == nil {
		return cmdOutput, nil // Successfully executed
	}

	// If execution with both pods fails, print error message
	fmt.Println("Exec Failed. Please put silence manually")
	return "", err
}

func ExecWithPod(kubeconfig *rest.Config, clientset *kubernetes.Clientset, podName string, cmd []string) (string, error) {
	req := clientset.CoreV1().RESTClient().Post().Resource("pods").Name(podName).
		Namespace(AccountNamespace).SubResource("exec")
	option := &corev1.PodExecOptions{
		Container: ContainerName,
		Command:   cmd,
		Stdin:     false,
		Stdout:    true,
		Stderr:    true,
		TTY:       false,
	}

	req.VersionedParams(option, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(kubeconfig, "POST", req.URL())
	if err != nil {
		return "", fmt.Errorf("failed to create executor: %w", err)
	}

	capture := &cluster.LogCapture{}
	errorCapture := &cluster.LogCapture{}
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:  nil,
		Stdout: capture,
		Stderr: errorCapture,
		Tty:    false,
	})
	if err != nil {
		return "", fmt.Errorf("failed to stream with context: %w", err)
	}

	return capture.GetStdOut(), nil
}
