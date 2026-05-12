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

package controller

import (
	"fmt"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ControllerTest struct {
	Name            string
	NamespacePrefix string
	Contexts        []ControllerTestContext
	BeforeEach      func() // optional
	AfterEach       func() // optional
}

func (c ControllerTest) Run() bool {
	return Describe(c.Name, func() {
		if f := c.BeforeEach; f != nil {
			BeforeEach(func() {
				f()
			})
		}

		for i, x := range c.Contexts {
			x.Run(fmt.Sprintf("%s-%d", c.NamespacePrefix, i))
		}

		if f := c.AfterEach; f != nil {
			AfterEach(func() {
				f()
			})
		}
	})
}

type ControllerTestContext struct {
	Name       string
	Test       func(namespace string)
	BeforeEach func(namespace string) // optional
	AfterEach  func(namespace string) // optional
}

func (c ControllerTestContext) Run(namespaceName string) {
	Context(c.Name, func() {
		namespace := &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespaceName,
			},
		}
		BeforeEach(func() {
			Expect(k8sClient.Create(ctx, namespace)).To(Succeed())
			if f := c.BeforeEach; f != nil {
				f(namespace.Name)
			}
		})

		c.Test(namespace.Name)

		AfterEach(func() {
			if f := c.AfterEach; f != nil {
				f(namespace.Name)
			}
			Expect(k8sClient.Delete(ctx, namespace)).To(Succeed())
		})
	})
}
