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
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	LabelNodeIndex = "nodeIndex"
)

type Suite struct {
	Name            string
	NamespacePrefix string
	KustomizeRoot   string
	Testcases       []Testcase
}

func (s Suite) namespace(i int) string {
	return fmt.Sprintf("%s-%d", s.NamespacePrefix, i)
}

func (s Suite) Run() {
	Context(s.Name, Ordered, func() {
		BeforeAll(func() {
			s.InitNode()
			s.Delete(true)
			s.Apply()
		})
		AfterAll(func() {
			s.Delete(false)
		})

		s.BeforeAddNode()
		s.AddNode()
		s.AfterAddNodeInit()
		s.AfterAddNode()
		s.RemoveNode()
		s.AfterRemoveNodeInit()
		s.AfterRemoveNode()
	})
}

func (s Suite) RemoveNode() {
	It("RemoveNode", func() {
		By("Stop worker node 1")
		_, err := Run(ClusterCmd("stop", "worker", "1"))
		Expect(err).To(Succeed())
		By("Ensure only 1 worker node is running")
		Eventually(func(g Gomega) {
			nodeNames, err := ListNodeNames()
			g.Expect(err).To(Succeed())
			g.Expect(nodeNames).To(HaveLen(1))
		}).To(Succeed())
	})
}

func (s Suite) InitNode() {
	By("Add nodeIndex label to the worker node")
	_, err := Run(KubectlCmd("label", "node", NodeName(0), LabelNodeIndex+"=0"))
	Expect(err).To(Succeed())
}

func (s Suite) AddNode() {
	It("AddNode", func() {
		By("Start worker node 1")
		_, err := Run(ClusterCmd("start", "worker", "1"))
		Expect(err).To(Succeed())
		By("Ensure 2 worker nodes are running")
		Eventually(func(g Gomega) {
			nodeNames, err := ListNodeNames()
			g.Expect(err).To(Succeed())
			g.Expect(nodeNames).To(HaveLen(2))
		}).To(Succeed())
		By("Add nodeIndex label to the worker node 1")
		_, err = Run(KubectlCmd("label", "node", NodeName(1), LabelNodeIndex+"=1"))
		Expect(err).To(Succeed())
		By("Load images to worker nodes")
		_, err = Run(exec.Command("make", "load"))
		Expect(err).To(Succeed())
	})
}

func (s Suite) Apply() {
	for i, tc := range s.Testcases {
		ns := s.namespace(i)
		By(fmt.Sprintf("Apply manifest for %s into %s", tc.Name, ns))
		_, err := Run(KubectlCmd("create", "namespace", ns))
		Expect(err).To(Succeed())
		Expect(tc.Args(ns, s.KustomizeRoot).Apply()).To(Succeed())
	}
}

func (s Suite) Delete(wait bool) {
	for i, tc := range s.Testcases {
		ns := s.namespace(i)
		By(fmt.Sprintf("Delete manifest for %s from %s", tc.Name, ns))
		Expect(tc.Args(ns, s.KustomizeRoot).Delete(wait)).To(Succeed())
	}
	for i := range s.Testcases {
		ns := s.namespace(i)
		By(fmt.Sprintf("Delete namespace %s", ns))
		_, err := Run(KubectlCmd("delete", "namespace", ns, "--ignore-not-found=true", fmt.Sprintf("--wait=%v", wait)))
		Expect(err).To(Succeed())
	}
}

func (s Suite) generateContextFromTestcases(name string, hookMapper func(Testcase) func(*TestcaseArgs)) {
	Context(name, func() {
		for i, tc := range s.Testcases {
			if f := hookMapper(tc); f != nil {
				Context(tc.Name, func() {
					f(tc.Args(s.namespace(i), s.KustomizeRoot))
				})
			}
		}
	})
}

func (s Suite) BeforeAddNode() {
	s.generateContextFromTestcases("BeforeAddNode", func(x Testcase) func(*TestcaseArgs) {
		return x.BeforeAddNode
	})
}

func (s Suite) AfterAddNode() {
	s.generateContextFromTestcases("AfterAddNode", func(x Testcase) func(*TestcaseArgs) {
		return x.AfterAddNode
	})
}

func (s Suite) AfterRemoveNode() {
	s.generateContextFromTestcases("AfterRemoveNode", func(x Testcase) func(*TestcaseArgs) {
		return x.AfterRemoveNode
	})
}

func (s Suite) AfterAddNodeInit() {
	s.generateContextFromTestcases("AfterAddNodeInit", func(x Testcase) func(*TestcaseArgs) {
		return x.AfterAddNodeInit
	})
}

func (s Suite) AfterRemoveNodeInit() {
	s.generateContextFromTestcases("AfterRemoveNodeInit", func(x Testcase) func(*TestcaseArgs) {
		return x.AfterRemoveNodeInit
	})
}

type Testcase struct {
	Name                string
	KustomizeDir        string
	BeforeAddNode       func(a *TestcaseArgs)
	AfterAddNodeInit    func(a *TestcaseArgs)
	AfterAddNode        func(a *TestcaseArgs)
	AfterRemoveNodeInit func(a *TestcaseArgs)
	AfterRemoveNode     func(a *TestcaseArgs)
}

func (tc Testcase) Args(namespace, kustomizeRoot string) *TestcaseArgs {
	return &TestcaseArgs{
		Namespace: namespace,
		Manifest:  filepath.Join(kustomizeRoot, tc.KustomizeDir),
	}
}

type TestcaseArgs struct {
	Namespace string
	Manifest  string
}

func (a TestcaseArgs) Apply() error {
	_, err := Run(KubectlCmd("-n", a.Namespace, "apply", "-k", a.Manifest))
	return err
}

func (a TestcaseArgs) Delete(wait bool) error {
	_, err := Run(KubectlCmd("-n", a.Namespace, "delete", "-k", a.Manifest, "--ignore-not-found=true", fmt.Sprintf("--wait=%v", wait)))
	return err
}

func (a TestcaseArgs) Reapply() error {
	if err := a.Delete(true); err != nil {
		return err
	}
	return a.Apply()
}
