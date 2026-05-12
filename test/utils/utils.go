/*
Copyright 2026 berquerant.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package utils

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/berquerant/daemonjob/internal/controller"
	. "github.com/onsi/ginkgo/v2" // nolint:revive,staticcheck
)

const (
	certmanagerVersion = "v1.20.2"
	certmanagerURLTmpl = "https://github.com/cert-manager/cert-manager/releases/download/%s/cert-manager.yaml"
)

func warnError(err error) {
	_, _ = fmt.Fprintf(GinkgoWriter, "warning: %v\n", err)
}

// Run executes the provided command within this context
func Run(cmd *exec.Cmd) (string, error) {
	dir, _ := GetProjectDir()
	cmd.Dir = dir

	if err := os.Chdir(cmd.Dir); err != nil {
		_, _ = fmt.Fprintf(GinkgoWriter, "chdir dir: %q\n", err)
	}

	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	command := strings.Join(cmd.Args, " ")
	_, _ = fmt.Fprintf(GinkgoWriter, "running: %q\n", command)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("%q failed with error %q: %w", command, string(output), err)
	}

	return string(output), nil
}

// ClusterCmd creates a command to run cluster.sh.
func ClusterCmd(arg ...string) *exec.Cmd {
	dir, _ := GetProjectDir()
	return exec.Command(filepath.Join(dir, "hack", "cluster.sh"), arg...)
}

func KubectlCmd(arg ...string) *exec.Cmd {
	dir, _ := GetProjectDir()
	return exec.Command(filepath.Join(dir, "hack", "tool.sh"), append([]string{"kubectl"}, arg...)...)
}

func Must[T any](t T, err error) T {
	if err != nil {
		panic(err)
	}
	return t
}

// UninstallCertManager uninstalls the cert manager
func UninstallCertManager() {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := KubectlCmd("delete", "-f", url)
	if _, err := Run(cmd); err != nil {
		warnError(err)
	}

	// Delete leftover leases in kube-system (not cleaned by default)
	kubeSystemLeases := []string{
		"cert-manager-cainjector-leader-election",
		"cert-manager-controller",
	}
	for _, lease := range kubeSystemLeases {
		cmd = KubectlCmd("delete", "lease", lease,
			"-n", "kube-system", "--ignore-not-found", "--force", "--grace-period=0")
		if _, err := Run(cmd); err != nil {
			warnError(err)
		}
	}
}

// InstallCertManager installs the cert manager bundle.
func InstallCertManager() error {
	url := fmt.Sprintf(certmanagerURLTmpl, certmanagerVersion)
	cmd := KubectlCmd("apply", "-f", url)
	if _, err := Run(cmd); err != nil {
		return err
	}
	// Wait for cert-manager-webhook to be ready, which can take time if cert-manager
	// was re-installed after uninstalling on a cluster.
	cmd = KubectlCmd("wait", "deployment.apps/cert-manager-webhook",
		"--for", "condition=Available",
		"--namespace", "cert-manager",
		"--timeout", "5m",
	)

	_, err := Run(cmd)
	return err
}

// IsCertManagerCRDsInstalled checks if any Cert Manager CRDs are installed
// by verifying the existence of key CRDs related to Cert Manager.
func IsCertManagerCRDsInstalled() bool {
	// List of common Cert Manager CRDs
	certManagerCRDs := []string{
		"certificates.cert-manager.io",
		"issuers.cert-manager.io",
		"clusterissuers.cert-manager.io",
		"certificaterequests.cert-manager.io",
		"orders.acme.cert-manager.io",
		"challenges.acme.cert-manager.io",
	}

	// Execute the kubectl command to get all CRDs
	cmd := KubectlCmd("get", "crds")
	output, err := Run(cmd)
	if err != nil {
		return false
	}

	// Check if any of the Cert Manager CRDs are present
	crdList := GetNonEmptyLines(output)
	for _, crd := range certManagerCRDs {
		for _, line := range crdList {
			if strings.Contains(line, crd) {
				return true
			}
		}
	}

	return false
}

// LoadImageToCluster loads a local docker image to the cluster
func LoadImageToCluster(name string) error {
	cmd := ClusterCmd("load", name)
	_, err := Run(cmd)
	return err
}

// GetNonEmptyLines converts given command output string into individual objects
// according to line breakers, and ignores the empty elements in it.
func GetNonEmptyLines(output string) []string {
	var res []string
	elements := strings.SplitSeq(output, "\n")
	for element := range elements {
		if element != "" {
			res = append(res, element)
		}
	}

	return res
}

// GetProjectDir will return the directory where the project is
func GetProjectDir() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return wd, fmt.Errorf("failed to get current working directory: %w", err)
	}
	wd = strings.ReplaceAll(wd, "/test/e2e", "")
	return wd, nil
}

// UncommentCode searches for target in the file and remove the comment prefix
// of the target content. The target content may span multiple lines.
func UncommentCode(filename, target, prefix string) error {
	// false positive
	// nolint:gosec
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read file %q: %w", filename, err)
	}
	strContent := string(content)

	idx := strings.Index(strContent, target)
	if idx < 0 {
		return fmt.Errorf("unable to find the code %q to be uncommented", target)
	}

	out := new(bytes.Buffer)
	_, err = out.Write(content[:idx])
	if err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	scanner := bufio.NewScanner(bytes.NewBufferString(target))
	if !scanner.Scan() {
		return nil
	}
	for {
		if _, err = out.WriteString(strings.TrimPrefix(scanner.Text(), prefix)); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
		// Avoid writing a newline in case the previous line was the last in target.
		if !scanner.Scan() {
			break
		}
		if _, err = out.WriteString("\n"); err != nil {
			return fmt.Errorf("failed to write to output: %w", err)
		}
	}

	if _, err = out.Write(content[idx+len(target):]); err != nil {
		return fmt.Errorf("failed to write to output: %w", err)
	}

	// false positive
	// nolint:gosec
	if err = os.WriteFile(filename, out.Bytes(), 0644); err != nil {
		return fmt.Errorf("failed to write file %q: %w", filename, err)
	}

	return nil
}

func GetDaemonJobState(namespace, name string) (string, error) {
	return Run(KubectlCmd("-n", namespace, "get", "daemonjob", name, "-o=jsonpath='{.status.state}'"))
}

func LinesIntoSlices(s string) []string {
	v := strings.TrimSpace(s)
	if v == "" {
		return nil
	}
	return strings.Split(v, "\n")
}

func GetDaemonJobWorkerJobs(namespace, daemonJobName string) ([]string, error) {
	return GetNamesByLabelSelector(namespace, "job",
		controller.DaemonJobLabelDaemonJobName, daemonJobName,
		controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
	)
}

func GetDaemonCronJobWorkerJobs(namespace, daemonCronJobName string) ([]string, error) {
	return GetNamesByLabelSelector(namespace, "job",
		controller.DaemonJobLabelDaemonCronJobName, daemonCronJobName,
		controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
	)
}

func GetDaemonCronJobBroadcastJobs(namespace, daemonCronJobName string) ([]string, error) {
	return GetNamesByLabelSelector(namespace, "job",
		controller.DaemonJobLabelDaemonCronJobName, daemonCronJobName,
		controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleBroadcast,
	)
}

func GetDaemonCronJobBroadcastCronJobs(namespace, daemonCronJobName string) ([]string, error) {
	return GetNamesByLabelSelector(namespace, "cronjob",
		controller.DaemonJobLabelDaemonCronJobName, daemonCronJobName,
		controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleBroadcast,
	)
}

func GetDaemonCronJobSetCronJobs(namespace, daemonCronJobSetName string) ([]string, error) {
	return GetNamesByLabelSelector(namespace, "cronjob",
		controller.DaemonJobLabelDaemonCronJobSetName, daemonCronJobSetName,
		controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
	)
}

func GetDaemonCronJobSetCronJobNodes(namespace, daemonCronJobSetName string) ([]string, error) {
	cronJobNames, err := GetDaemonCronJobSetCronJobs(namespace, daemonCronJobSetName)
	if err != nil {
		return nil, err
	}
	nodeNames := make([]string, len(cronJobNames))
	for i, cronJob := range cronJobNames {
		out, err := Run(KubectlCmd("-n", namespace, "get", "cronjob", cronJob, "-o=jsonpath={.metadata.labels}"))
		if err != nil {
			return nil, err
		}
		labels := map[string]string{}
		if err := json.Unmarshal([]byte(out), &labels); err != nil {
			return nil, err
		}
		nodeNames[i] = labels[controller.DaemonJobLabelNode]
	}
	sort.Strings(nodeNames)
	return nodeNames, nil
}

func GetDaemonCronJobSetJobsByNode(namespace, daemonCronJobSetName, nodeName string) ([]string, error) {
	return GetNamesByLabelSelector(namespace, "job",
		controller.DaemonJobLabelDaemonCronJobSetName, daemonCronJobSetName,
		controller.DaemonJobLabelRole, controller.DaemonJobLabelRoleWorker,
		controller.DaemonJobLabelNode, nodeName,
	)
}

func GetNamesByLabelSelector(namespace, kind string, labelSelectorKeyValue ...string) ([]string, error) {
	var arg []string
	if namespace != "" {
		arg = append(arg, "-n", namespace)
	}
	arg = append(arg, "get", kind, "-o=custom-columns=NAME:.metadata.name", "--no-headers")
	if len(labelSelectorKeyValue) > 0 {
		xs := make([]string, len(labelSelectorKeyValue)/2)
		for i := range xs {
			b := 2 * i
			k := labelSelectorKeyValue[b]
			v := labelSelectorKeyValue[b+1]
			xs[i] = k + "=" + v
		}
		arg = append(arg, "-l", strings.Join(xs, ","))
	}
	out, err := Run(KubectlCmd(arg...))
	if err != nil {
		return nil, err
	}
	if strings.Contains(out, "No resources found") {
		return nil, nil
	}
	return LinesIntoSlices(out), nil
}

func GetNodeNames(labelSelectorKeyValue ...string) ([]string, error) {
	return GetNamesByLabelSelector("", "node", labelSelectorKeyValue...)
}

func Retry(f func() error) error {
	var err error
	for i := range 60 {
		if i > 0 {
			time.Sleep(time.Second)
		}
		if err = f(); err == nil {
			return nil
		}
	}
	return err
}

func GetJobPodNames(namespace, name string) ([]string, error) {
	out, err := Run(KubectlCmd("-n", namespace, "get", "job", name, "-o=jsonpath={.spec.template.metadata.labels}"))
	if err != nil {
		return nil, err
	}
	labels := map[string]string{}
	if err := json.Unmarshal([]byte(out), &labels); err != nil {
		return nil, err
	}
	uid, ok := labels["batch.kubernetes.io/controller-uid"]
	if !ok {
		return nil, errors.New("Job has no label batch.kubernetes.io/controller-uid")
	}
	return GetNamesByLabelSelector(namespace, "pod", "batch.kubernetes.io/controller-uid", uid)
}

func NodeName(nodeIndex int) string {
	return "daemonjob-k0s-worker-" + strconv.Itoa(nodeIndex)
}

func ListNodeNames() ([]string, error) {
	out, err := Run(KubectlCmd("get", "node", "-o=custom-columns=Name:.metadata.name", "--no-headers"))
	if err != nil {
		return nil, err
	}
	return LinesIntoSlices(out), nil
}

func GetJobState(namespace, name, state string) (bool, error) {
	status, err := Run(KubectlCmd("-n", namespace, "get", "job", name,
		fmt.Sprintf("-o=jsonpath={.status.conditions[?(@.type=='%s')].status}", state),
	))
	if err != nil {
		return false, err
	}
	return status == "True", nil
}
