// Copyright 2026 berquerant.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"fmt"
	"sort"
	"strconv"

	. "github.com/onsi/gomega"
)

func EventuallyDaemonJobState(namespace, name, state string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		state, err := GetDaemonJobState(namespace, name)
		g.Expect(err).To(Succeed())
		g.Expect(state).To(Equal(state))
	})
}

func EventualyDaemonCronJobState(namespace, name, state string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		state, err := Run(KubectlCmd("-n", namespace, "get", "daemoncronjob", name,
			fmt.Sprintf("-o=jsonpath={.status.conditions[?(@.type=='%s')].status}", state),
		))
		g.Expect(err).To(Succeed())
		g.Expect(state).To(Equal("True"))
	})
}

func EventuallyDaemonCronJobSetState(namespace, name, state string, desired, available int) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		state, err := Run(KubectlCmd("-n", namespace, "get", "daemoncronjobset", name,
			fmt.Sprintf("-o=jsonpath={.status.conditions[?(@.type=='%s')].status}", state),
		))
		g.Expect(err).To(Succeed())
		g.Expect(state).To(Equal("True"))

		d, err := Run(KubectlCmd("-n", namespace, "get", "daemoncronjobset", name,
			"-o=jsonpath={.status.numberDesired}",
		))
		g.Expect(err).To(Succeed())
		di, err := strconv.Atoi(d)
		g.Expect(err).To(Succeed())
		g.Expect(di).To(Equal(desired))

		a, err := Run(KubectlCmd("-n", namespace, "get", "daemoncronjobset", name,
			"-o=jsonpath={.status.numberAvailable}",
		))
		g.Expect(err).To(Succeed())
		ai, err := strconv.Atoi(a)
		g.Expect(err).To(Succeed())
		g.Expect(ai).To(Equal(available))
	})
}

func EventuallyDaemonCronJobSetCronJobNodes(namespace, name string, node ...string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		nodeNames, err := GetDaemonCronJobSetCronJobNodes(namespace, name)
		g.Expect(err).To(Succeed())
		sort.Strings(node)
		g.Expect(nodeNames).To(Equal(node))
	})
}

func EventuallyJobState(namespace, name, state string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		status, err := Run(KubectlCmd("-n", namespace, "get", "job", name,
			fmt.Sprintf("-o=jsonpath={.status.conditions[?(@.type=='%s')].status}", state),
		))
		g.Expect(err).To(Succeed())
		g.Expect(status).To(Equal("True"))
	})
}

func EventuallyJobComplete(namespace, name string) AsyncAssertion {
	return EventuallyJobState(namespace, name, "Complete")
}

func EventuallyNotFound(namespace, name, kind string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		var arg []string
		if namespace != "" {
			arg = append(arg, "-n", namespace)
		}
		arg = append(arg, "get", kind, name)
		out, _ := Run(KubectlCmd(arg...))
		g.Expect(out).To(ContainSubstring("NotFound"))
	})
}

func EventuallyFound(namespace, name, kind string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		var arg []string
		if namespace != "" {
			arg = append(arg, "-n", namespace)
		}
		arg = append(arg, "get", kind, name)
		_, err := Run(KubectlCmd(arg...))
		g.Expect(err).To(Succeed())
	})
}

func EventuallyNotFoundByLabelSelector(namespace, kind string, labelSelectorKeyValue ...string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		out, err := GetNamesByLabelSelector(namespace, kind, labelSelectorKeyValue...)
		g.Expect(err).To(Succeed())
		g.Expect(out).To(BeEmpty())
	})
}

func EventuallyFoundByLabelSelector(namespace, kind string, labelSelectorKeyValue ...string) AsyncAssertion {
	return EventuallyWithOffset(1, func(g Gomega) {
		out, err := GetNamesByLabelSelector(namespace, kind, labelSelectorKeyValue...)
		g.Expect(err).To(Succeed())
		g.Expect(out).NotTo(BeEmpty())
	})
}
