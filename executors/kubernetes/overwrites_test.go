package kubernetes

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"gitlab.com/gitlab-org/gitlab-runner/common"
)

type variableOverwrites map[string]string

func buildOverwriteVariables(overwrites variableOverwrites, podAnnotations map[string]string) common.JobVariables {
	variables := make(common.JobVariables, 8)

	for variableKey, overwriteValue := range overwrites {
		if overwriteValue != "" {
			variables = append(variables, common.JobVariable{Key: variableKey, Value: overwriteValue})
		}
	}

	for k, v := range podAnnotations {
		variables = append(variables, common.JobVariable{Key: k, Value: v})
	}

	return variables
}

func stdoutLogger() common.BuildLogger {
	return common.NewBuildLogger(&common.Trace{Writer: os.Stdout}, logrus.WithFields(logrus.Fields{}))
}

func TestOverwrites(t *testing.T) {
	logger := stdoutLogger()
	overwritesAllowedConfig := &common.KubernetesConfig{
		NamespaceOverwriteAllowed:      ".*",
		ServiceAccountOverwriteAllowed: ".*",
		BearerTokenOverwriteAllowed:    true,
		PodAnnotationsOverwriteAllowed: ".*",
		PodAnnotations: map[string]string{
			"test1":                     "test1",
			"test2":                     "test2",
			"test3":                     "test3",
			"org.gitlab/runner-version": "v10.4.0",
			"org.gitlab/gitlab-host":    "https://gitlab.example.com",
			"iam.amazonaws.com/role":    "arn:aws:iam::123456789012:role/",
		},
		CPULimit:                         "5",
		CPURequest:                       "3",
		CPULimitOverwriteMaxAllowed:      "10",
		CPURequestOverwriteMaxAllowed:    "8",
		MemoryLimit:                      "5Gi",
		MemoryRequest:                    "2Gi",
		MemoryLimitOverwriteMaxAllowed:   "15Gi",
		MemoryRequestOverwriteMaxAllowed: "10Gi",
	}

	//nolint:lll
	tests := []struct {
		Name                                 string
		Config                               *common.KubernetesConfig
		NamespaceOverwriteVariableValue      string
		ServiceAccountOverwriteVariableValue string
		BearerTokenOverwriteVariableValue    string
		PodAnnotationsOverwriteValues        map[string]string
		CPULimitOverwriteVariableValue       string
		CPURequestOverwriteVariableValue     string
		MemoryLimitOverwriteVariableValue    string
		MemoryRequestOverwriteVariableValue  string
		Expected                             *overwrites
		Error                                error
	}{
		{
			Name:     "Empty Configuration",
			Config:   &common.KubernetesConfig{},
			Expected: &overwrites{},
		},
		{
			Name:                                 "All overwrites allowed",
			Config:                               overwritesAllowedConfig,
			NamespaceOverwriteVariableValue:      "my_namespace",
			ServiceAccountOverwriteVariableValue: "my_service_account",
			BearerTokenOverwriteVariableValue:    "my_bearer_token",
			PodAnnotationsOverwriteValues: map[string]string{
				"KUBERNETES_POD_ANNOTATIONS_1":            "test3=test3=1",
				"KUBERNETES_POD_ANNOTATIONS_2":            "test4=test4",
				"KUBERNETES_POD_ANNOTATIONS_gilabversion": "org.gitlab/runner-version=v10.4.0-override",
				"KUBERNETES_POD_ANNOTATIONS_kube2iam":     "iam.amazonaws.com/role=arn:aws:iam::kjcbs;dkjbck=jxzweopiu:role/",
			},
			CPULimitOverwriteVariableValue:      "10",
			CPURequestOverwriteVariableValue:    "8",
			MemoryLimitOverwriteVariableValue:   "15Gi",
			MemoryRequestOverwriteVariableValue: "10Gi",
			Expected: &overwrites{
				namespace:      "my_namespace",
				serviceAccount: "my_service_account",
				bearerToken:    "my_bearer_token",
				podAnnotations: map[string]string{
					"test1":                     "test1",
					"test2":                     "test2",
					"test3":                     "test3=1",
					"test4":                     "test4",
					"org.gitlab/runner-version": "v10.4.0-override",
					"org.gitlab/gitlab-host":    "https://gitlab.example.com",
					"iam.amazonaws.com/role":    "arn:aws:iam::kjcbs;dkjbck=jxzweopiu:role/",
				},
				cpuLimit:      "10",
				cpuRequest:    "8",
				memoryLimit:   "15Gi",
				memoryRequest: "10Gi",
			},
		},
		{
			Name: "No overwrites allowed",
			Config: &common.KubernetesConfig{
				Namespace:      "my_namespace",
				ServiceAccount: "my_service_account",
				BearerToken:    "my_bearer_token",
				PodAnnotations: map[string]string{
					"test1": "test1",
					"test2": "test2",
				},
				CPULimit:      "1",
				CPURequest:    "1",
				MemoryLimit:   "2Gi",
				MemoryRequest: "2Gi",
			},
			NamespaceOverwriteVariableValue:      "another_namespace",
			ServiceAccountOverwriteVariableValue: "another_service_account",
			BearerTokenOverwriteVariableValue:    "another_bearer_token",
			PodAnnotationsOverwriteValues: map[string]string{
				"KUBERNETES_POD_ANNOTATIONS_1": "test3=test3",
				"KUBERNETES_POD_ANNOTATIONS_2": "test4=test4",
			},
			CPULimitOverwriteVariableValue:      "10",
			CPURequestOverwriteVariableValue:    "8",
			MemoryLimitOverwriteVariableValue:   "15Gi",
			MemoryRequestOverwriteVariableValue: "10Gi",
			Expected: &overwrites{
				namespace:      "my_namespace",
				serviceAccount: "my_service_account",
				bearerToken:    "my_bearer_token",
				podAnnotations: map[string]string{
					"test1": "test1",
					"test2": "test2",
				},
				cpuLimit:      "1",
				cpuRequest:    "1",
				memoryLimit:   "2Gi",
				memoryRequest: "2Gi",
			},
		},
		{
			Name: "Resource overwrites the same",
			Config: &common.KubernetesConfig{
				CPURequestOverwriteMaxAllowed:    "10",
				CPULimitOverwriteMaxAllowed:      "12",
				MemoryRequestOverwriteMaxAllowed: "10",
				MemoryLimitOverwriteMaxAllowed:   "12",
			},
			CPURequestOverwriteVariableValue:    "10",
			CPULimitOverwriteVariableValue:      "12",
			MemoryRequestOverwriteVariableValue: "10",
			MemoryLimitOverwriteVariableValue:   "12",
			Expected: &overwrites{
				cpuRequest:    "10",
				cpuLimit:      "12",
				memoryRequest: "10",
				memoryLimit:   "12",
			},
		},
		{
			Name: "Namespace failure",
			Config: &common.KubernetesConfig{
				NamespaceOverwriteAllowed: "not-a-match",
			},
			NamespaceOverwriteVariableValue: "my_namespace",
			Error:                           new(malformedOverwriteError),
		},
		{
			Name: "ServiceAccount failure",
			Config: &common.KubernetesConfig{
				ServiceAccountOverwriteAllowed: "not-a-match",
			},
			ServiceAccountOverwriteVariableValue: "my_service_account",
			Error:                                new(malformedOverwriteError),
		},
		{
			Name: "PodAnnotations failure",
			Config: &common.KubernetesConfig{
				PodAnnotationsOverwriteAllowed: "not-a-match",
			},
			PodAnnotationsOverwriteValues: map[string]string{
				"KUBERNETES_POD_ANNOTATIONS_1": "test1=test1",
			},
			Error: new(malformedOverwriteError),
		},
		{
			Name: "PodAnnotations malformed key",
			Config: &common.KubernetesConfig{
				PodAnnotationsOverwriteAllowed: ".*",
			},
			PodAnnotationsOverwriteValues: map[string]string{
				"KUBERNETES_POD_ANNOTATIONS_1": "test1",
			},
			Error: new(malformedOverwriteError),
		},
		{
			Name: "CPULimit too high",
			Config: &common.KubernetesConfig{
				CPULimitOverwriteMaxAllowed: "10",
			},
			CPULimitOverwriteVariableValue: "12",
			Error:                          new(overwriteTooHighError),
		},
		{
			Name: "CPULimit too high using millicpu",
			Config: &common.KubernetesConfig{
				CPULimitOverwriteMaxAllowed: "500m",
			},
			CPULimitOverwriteVariableValue: "600m",
			Error:                          new(overwriteTooHighError),
		},
		{
			Name: "CPURequest too high",
			Config: &common.KubernetesConfig{
				CPURequestOverwriteMaxAllowed: "10",
			},
			CPURequestOverwriteVariableValue: "12",
			Error:                            new(overwriteTooHighError),
		},
		{
			Name: "CPURequest too high using millicpu",
			Config: &common.KubernetesConfig{
				CPURequestOverwriteMaxAllowed: "500m",
			},
			CPURequestOverwriteVariableValue: "600m",
			Error:                            new(overwriteTooHighError),
		},
		{
			Name: "MemoryLimit too high",
			Config: &common.KubernetesConfig{
				MemoryLimitOverwriteMaxAllowed: "2Gi",
			},
			MemoryLimitOverwriteVariableValue: "10Gi",
			Error:                             new(overwriteTooHighError),
		},
		{
			Name: "MemoryLimit too high Mi",
			Config: &common.KubernetesConfig{
				MemoryLimitOverwriteMaxAllowed: "20Mi",
			},
			MemoryLimitOverwriteVariableValue: "10Gi",
			Error:                             new(overwriteTooHighError),
		},
		{
			Name: "MemoryRequest too high",
			Config: &common.KubernetesConfig{
				MemoryRequestOverwriteMaxAllowed: "2Gi",
			},
			MemoryRequestOverwriteVariableValue: "10Gi",
			Error:                               new(overwriteTooHighError),
		},
		{
			Name: "MemoryRequest too high Mi",
			Config: &common.KubernetesConfig{
				MemoryRequestOverwriteMaxAllowed: "20Mi",
			},
			MemoryRequestOverwriteVariableValue: "100Mi",
			Error:                               new(overwriteTooHighError),
		},
		{
			Name: "MemoryRequest too high different suffix",
			Config: &common.KubernetesConfig{
				MemoryRequestOverwriteMaxAllowed: "2Gi",
			},
			MemoryRequestOverwriteVariableValue: "5000Mi",
			Error:                               new(overwriteTooHighError),
		},
	}

	for _, test := range tests {
		t.Run(test.Name, func(t *testing.T) {
			variables := buildOverwriteVariables(
				variableOverwrites{
					NamespaceOverwriteVariableName:      test.NamespaceOverwriteVariableValue,
					ServiceAccountOverwriteVariableName: test.ServiceAccountOverwriteVariableValue,
					BearerTokenOverwriteVariableValue:   test.BearerTokenOverwriteVariableValue,
					CPULimitOverwriteVariableValue:      test.CPULimitOverwriteVariableValue,
					CPURequestOverwriteVariableValue:    test.CPURequestOverwriteVariableValue,
					MemoryLimitOverwriteVariableValue:   test.MemoryLimitOverwriteVariableValue,
					MemoryRequestOverwriteVariableValue: test.MemoryRequestOverwriteVariableValue,
				},
				test.PodAnnotationsOverwriteValues,
			)

			values, err := createOverwrites(test.Config, variables, logger)
			assert.True(t, errors.Is(err, test.Error), "expected err %T, but got %T", test.Error, err)
			assert.Equal(t, test.Expected, values)
		})
	}
}

func Test_overwriteTooHighError_Is(t *testing.T) {
	tests := []struct {
		err        error
		expectedIs bool
	}{
		{
			err:        errors.New("false"),
			expectedIs: false,
		},
		{
			err:        new(emptyTestError),
			expectedIs: false,
		},
		{
			err:        new(overwriteTooHighError),
			expectedIs: true,
		},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%T", tt.err), func(t *testing.T) {
			err := overwriteTooHighError{}
			assert.Equal(t, tt.expectedIs, err.Is(tt.err))
		})
	}
}

type emptyTestError struct{}

func (e *emptyTestError) Error() string {
	return ""
}
