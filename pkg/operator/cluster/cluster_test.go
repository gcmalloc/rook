/*
Copyright 2016 The Rook Authors. All rights reserved.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

Some of the code below came from https://github.com/coreos/etcd-operator
which also has the apache 2.0 license.
*/
package cluster

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"

	"github.com/rook/rook/pkg/clusterd"
	"github.com/rook/rook/pkg/operator/k8sutil"
	testop "github.com/rook/rook/pkg/operator/test"
	exectest "github.com/rook/rook/pkg/util/exec/test"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateSecrets(t *testing.T) {
	clientset := testop.New(3)
	info := testop.CreateClusterInfo(1)
	executor := &exectest.MockExecutor{
		MockExecuteCommandWithOutputFile: func(actionName string, command string, outFileArg string, args ...string) (string, error) {
			return "{\"key\":\"mysecurekey\"}", nil
		},
	}
	c := &Cluster{Spec: Spec{VersionTag: "myversion"}}
	c.Name = "myrook"
	c.Namespace = "myns"
	configDir, _ := ioutil.TempDir("", "")
	defer os.RemoveAll(configDir)
	c.Init(&clusterd.Context{KubeContext: clusterd.KubeContext{Clientset: clientset}, ConfigDir: configDir, Executor: executor})
	defer os.RemoveAll(c.context.ConfigDir)

	err := c.createClientAccess(info)
	assert.Nil(t, err)

	secretName := fmt.Sprintf("%s-rook-user", c.Namespace)
	secret, err := clientset.CoreV1().Secrets(k8sutil.DefaultNamespace).Get(secretName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.Equal(t, secretName, secret.Name)
	assert.Equal(t, 1, len(secret.StringData))
}

func TestCreateInitialCrushMap(t *testing.T) {
	clientset := testop.New(3)
	executor := &exectest.MockExecutor{}
	c := &Cluster{}
	c.Namespace = "rook294"
	c.Init(&clusterd.Context{KubeContext: clusterd.KubeContext{Clientset: clientset}, Executor: executor})

	// create the initial crush map and verify that a configmap value was created that says the crush map was created
	err := c.createInitialCrushMap()
	assert.Nil(t, err)
	cm, err := clientset.CoreV1().ConfigMaps(c.Namespace).Get(crushConfigMapName, metav1.GetOptions{})
	assert.Nil(t, err)
	assert.NotNil(t, cm)
	assert.Equal(t, "1", cm.Data[crushmapCreatedKey])

	// the crushmap should NOT get created again
	executor.MockExecuteCommandWithOutputFile = func(actionName string, command string, outFileArg string, args ...string) (string, error) {
		return "", fmt.Errorf("crushmap was already created, we shouldn't be calling this again")
	}
	err = c.createInitialCrushMap()
	assert.Nil(t, err)
}
