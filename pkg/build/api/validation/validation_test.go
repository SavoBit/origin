package validation

import (
	"strings"
	"testing"

	kapi "k8s.io/kubernetes/pkg/api"
	"k8s.io/kubernetes/pkg/util/fielderrors"

	buildapi "github.com/openshift/origin/pkg/build/api"
)

func TestBuildValidationSuccess(t *testing.T) {
	build := &buildapi.Build{
		ObjectMeta: kapi.ObjectMeta{Name: "buildid", Namespace: "default"},
		Spec: buildapi.BuildSpec{
			Source: buildapi.BuildSource{
				Type: buildapi.BuildSourceGit,
				Git: &buildapi.GitBuildSource{
					URI: "http://github.com/my/repository",
				},
				ContextDir: "context",
			},
			Strategy: buildapi.BuildStrategy{
				Type:           buildapi.DockerBuildStrategyType,
				DockerStrategy: &buildapi.DockerBuildStrategy{},
			},
			Output: buildapi.BuildOutput{
				To: &kapi.ObjectReference{
					Kind: "DockerImage",
					Name: "repository/data",
				},
			},
		},
		Status: buildapi.BuildStatus{
			Phase: buildapi.BuildPhaseNew,
		},
	}
	if result := ValidateBuild(build); len(result) > 0 {
		t.Errorf("Unexpected validation error returned %v", result)
	}
}

func TestBuildValidationFailure(t *testing.T) {
	build := &buildapi.Build{
		ObjectMeta: kapi.ObjectMeta{Name: "", Namespace: ""},
		Spec: buildapi.BuildSpec{
			Source: buildapi.BuildSource{
				Type: "Git123",
				Git: &buildapi.GitBuildSource{
					URI: "http://github.com/my/repository",
				},
				ContextDir: "context",
			},
			Strategy: buildapi.BuildStrategy{
				Type:           buildapi.DockerBuildStrategyType,
				DockerStrategy: &buildapi.DockerBuildStrategy{},
			},
			Output: buildapi.BuildOutput{
				To: &kapi.ObjectReference{
					Kind: "DockerImage",
					Name: "repository/data",
				},
			},
		},
		Status: buildapi.BuildStatus{
			Phase: buildapi.BuildPhaseNew,
		},
	}
	if result := ValidateBuild(build); len(result) != 3 {
		t.Errorf("Unexpected validation result: %v", result)
	}
}

func newDefaultParameters() buildapi.BuildSpec {
	return buildapi.BuildSpec{
		Source: buildapi.BuildSource{
			Type: buildapi.BuildSourceGit,
			Git: &buildapi.GitBuildSource{
				URI: "http://github.com/my/repository",
			},
			ContextDir: "context",
		},
		Strategy: buildapi.BuildStrategy{
			Type:           buildapi.DockerBuildStrategyType,
			DockerStrategy: &buildapi.DockerBuildStrategy{},
		},
		Output: buildapi.BuildOutput{
			To: &kapi.ObjectReference{
				Kind: "DockerImage",
				Name: "repository/data",
			},
		},
	}
}

func newNonDefaultParameters() buildapi.BuildSpec {
	o := newDefaultParameters()
	o.Source.Git.URI = "changed"
	return o
}

func TestValidateBuildUpdate(t *testing.T) {
	old := &buildapi.Build{
		ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
		Spec:       newDefaultParameters(),
		Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseRunning},
	}

	errs := ValidateBuildUpdate(
		&buildapi.Build{
			ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
			Spec:       newDefaultParameters(),
			Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseComplete},
		},
		old,
	)
	if len(errs) != 0 {
		t.Errorf("expected success: %v", errs)
	}

	errorCases := map[string]struct {
		Old    *buildapi.Build
		Update *buildapi.Build
		T      fielderrors.ValidationErrorType
		F      string
	}{
		"changed spec": {
			Old: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
			},
			Update: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newNonDefaultParameters(),
			},
			T: fielderrors.ValidationErrorTypeInvalid,
			F: "spec",
		},
		"update from terminal1": {
			Old: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseComplete},
			},
			Update: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseRunning},
			},
			T: fielderrors.ValidationErrorTypeInvalid,
			F: "status.Phase",
		},
		"update from terminal2": {
			Old: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseCancelled},
			},
			Update: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseRunning},
			},
			T: fielderrors.ValidationErrorTypeInvalid,
			F: "status.Phase",
		},
		"update from terminal3": {
			Old: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseError},
			},
			Update: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseRunning},
			},
			T: fielderrors.ValidationErrorTypeInvalid,
			F: "status.Phase",
		},
		"update from terminal4": {
			Old: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseFailed},
			},
			Update: &buildapi.Build{
				ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault, Name: "my-build", ResourceVersion: "1"},
				Spec:       newDefaultParameters(),
				Status:     buildapi.BuildStatus{Phase: buildapi.BuildPhaseRunning},
			},
			T: fielderrors.ValidationErrorTypeInvalid,
			F: "status.Phase",
		},
	}

	for k, v := range errorCases {
		errs := ValidateBuildUpdate(v.Update, v.Old)
		if len(errs) == 0 {
			t.Errorf("expected failure %s for %v", k, v.Update)
			continue
		}
		for i := range errs {
			if errs[i].(*fielderrors.ValidationError).Type != v.T {
				t.Errorf("%s: expected errors to have type %s: %v", k, v.T, errs[i])
			}
			if errs[i].(*fielderrors.ValidationError).Field != v.F {
				t.Errorf("%s: expected errors to have field %s: %v", k, v.F, errs[i])
			}
		}
	}
}

func TestBuildConfigGitSourceWithProxyFailure(t *testing.T) {
	buildConfig := &buildapi.BuildConfig{
		ObjectMeta: kapi.ObjectMeta{Name: "config-id", Namespace: "namespace"},
		Spec: buildapi.BuildConfigSpec{
			BuildSpec: buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI:        "git://github.com/my/repository",
						HTTPProxy:  "127.0.0.1:3128",
						HTTPSProxy: "127.0.0.1:3128",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
	}
	errors := ValidateBuildConfig(buildConfig)
	if len(errors) != 1 {
		t.Errorf("Expected one error, got %d", len(errors))
	}
	err := errors[0].(*fielderrors.ValidationError)
	if err.Type != fielderrors.ValidationErrorTypeInvalid {
		t.Errorf("Expected invalid value validation error, got %q", err.Type)
	}
	if err.Detail != "only http:// and https:// GIT protocols are allowed with HTTP or HTTPS proxy set" {
		t.Errorf("Exptected git:// protocol with proxy validation error, got: %q", err.Detail)
	}
}

// TestBuildConfigDockerStrategyImageChangeTrigger ensures that it is invalid to
// have a BuildConfig with Docker strategy and an ImageChangeTrigger where
// neither DockerStrategy.From nor ImageChange.From are defined.
func TestBuildConfigDockerStrategyImageChangeTrigger(t *testing.T) {
	buildConfig := &buildapi.BuildConfig{
		ObjectMeta: kapi.ObjectMeta{Name: "config-id", Namespace: "namespace"},
		Spec: buildapi.BuildConfigSpec{
			BuildSpec: buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
			Triggers: []buildapi.BuildTriggerPolicy{
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
			},
		},
	}
	errors := ValidateBuildConfig(buildConfig)
	switch len(errors) {
	case 0:
		t.Errorf("Expected validation error, got nothing")
	case 1:
		err, ok := errors[0].(*fielderrors.ValidationError)
		if !ok {
			t.Fatalf("Expected error to be fielderrors.ValidationError, got %T", errors[0])
		}
		if err.Type != fielderrors.ValidationErrorTypeRequired {
			t.Errorf("Expected error to be '%v', got '%v'", fielderrors.ValidationErrorTypeRequired, err.Type)
		}
	default:
		t.Errorf("Expected a single validation error, got %v", errors)
	}
}

func TestBuildConfigValidationFailureRequiredName(t *testing.T) {
	buildConfig := &buildapi.BuildConfig{
		ObjectMeta: kapi.ObjectMeta{Name: "", Namespace: "foo"},
		Spec: buildapi.BuildConfigSpec{
			BuildSpec: buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
	}
	errors := ValidateBuildConfig(buildConfig)
	if len(errors) != 1 {
		t.Fatalf("Unexpected validation errors %v", errors)
	}
	err := errors[0].(*fielderrors.ValidationError)
	if err.Type != fielderrors.ValidationErrorTypeRequired {
		t.Errorf("Unexpected error type, expected %s, got %s", fielderrors.ValidationErrorTypeRequired, err.Type)
	}
	if err.Field != "metadata.name" {
		t.Errorf("Unexpected field name expected metadata.name, got %s", err.Field)
	}
}

func TestBuildConfigImageChangeTriggers(t *testing.T) {
	tests := []struct {
		name        string
		triggers    []buildapi.BuildTriggerPolicy
		expectError bool
		errorType   fielderrors.ValidationErrorType
	}{
		{
			name: "valid default trigger",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
			},
			expectError: false,
		},
		{
			name: "more than one default trigger",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
			},
			expectError: true,
			errorType:   fielderrors.ValidationErrorTypeInvalid,
		},
		{
			name: "missing image change struct",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type: buildapi.ImageChangeBuildTriggerType,
				},
			},
			expectError: true,
			errorType:   fielderrors.ValidationErrorTypeRequired,
		},
		{
			name: "only one default image change trigger",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind: "ImageStreamTag",
							Name: "myimage:tag",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "invalid reference kind for trigger",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind: "DockerImage",
							Name: "myimage:tag",
						},
					},
				},
			},
			expectError: true,
			errorType:   fielderrors.ValidationErrorTypeInvalid,
		},
		{
			name: "empty reference kind for trigger",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Name: "myimage:tag",
						},
					},
				},
			},
			expectError: true,
			errorType:   fielderrors.ValidationErrorTypeInvalid,
		},
		{
			name: "duplicate imagestreamtag references",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind: "ImageStreamTag",
							Name: "myimage:tag",
						},
					},
				},
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind: "ImageStreamTag",
							Name: "myimage:tag",
						},
					},
				},
			},
			expectError: true,
			errorType:   fielderrors.ValidationErrorTypeInvalid,
		},
		{
			name: "duplicate imagestreamtag - same as strategy ref",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type:        buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{},
				},
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind: "ImageStreamTag",
							Name: "builderimage:latest",
						},
					},
				},
			},
			expectError: true,
			errorType:   fielderrors.ValidationErrorTypeInvalid,
		},
		{
			name: "imagestreamtag references with same name, different ns",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      "myimage:tag",
							Namespace: "ns1",
						},
					},
				},
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      "myimage:tag",
							Namespace: "ns2",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "imagestreamtag references with same name, same ns",
			triggers: []buildapi.BuildTriggerPolicy{
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      "myimage:tag",
							Namespace: "ns",
						},
					},
				},
				{
					Type: buildapi.ImageChangeBuildTriggerType,
					ImageChange: &buildapi.ImageChangeTrigger{
						From: &kapi.ObjectReference{
							Kind:      "ImageStreamTag",
							Name:      "myimage:tag",
							Namespace: "ns",
						},
					},
				},
			},
			expectError: true,
			errorType:   fielderrors.ValidationErrorTypeInvalid,
		},
	}

	for _, tc := range tests {
		buildConfig := &buildapi.BuildConfig{
			ObjectMeta: kapi.ObjectMeta{Name: "bar", Namespace: "foo"},
			Spec: buildapi.BuildConfigSpec{
				BuildSpec: buildapi.BuildSpec{
					Source: buildapi.BuildSource{
						Type: buildapi.BuildSourceGit,
						Git: &buildapi.GitBuildSource{
							URI: "http://github.com/my/repository",
						},
						ContextDir: "context",
					},
					Strategy: buildapi.BuildStrategy{
						Type: buildapi.SourceBuildStrategyType,
						SourceStrategy: &buildapi.SourceBuildStrategy{
							From: kapi.ObjectReference{
								Kind: "ImageStreamTag",
								Name: "builderimage:latest",
							},
						},
					},
					Output: buildapi.BuildOutput{
						To: &kapi.ObjectReference{
							Kind: "DockerImage",
							Name: "repository/data",
						},
					},
				},
				Triggers: tc.triggers,
			},
		}
		errors := ValidateBuildConfig(buildConfig)
		// Check whether an error was returned
		if hasError := len(errors) > 0; hasError != tc.expectError {
			t.Errorf("%s: did not get expected result: %#v", tc.name, errors)
		}
		// Check whether it's the expected error type
		if len(errors) > 0 && tc.expectError && tc.errorType != "" {
			verr, ok := errors[0].(*fielderrors.ValidationError)
			if !ok {
				t.Errorf("%s: unexpected error: %#v. Expected ValidationError of type: %s", tc.name, errors[0], verr.Type)
				continue
			}
			if verr.Type != tc.errorType {
				t.Errorf("%s: unexpected error type. Expected: %s. Got: %s", tc.name, tc.errorType, verr.Type)
			}
		}
	}
}

func TestBuildConfigValidationOutputFailure(t *testing.T) {
	buildConfig := &buildapi.BuildConfig{
		ObjectMeta: kapi.ObjectMeta{Name: ""},
		Spec: buildapi.BuildConfigSpec{
			BuildSpec: buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Name: "other",
					},
				},
			},
		},
	}
	if result := ValidateBuildConfig(buildConfig); len(result) != 3 {
		for _, e := range result {
			t.Errorf("Unexpected validation result %v", e)
		}
	}
}

func TestValidateBuildRequest(t *testing.T) {
	testCases := map[string]*buildapi.BuildRequest{
		string(fielderrors.ValidationErrorTypeRequired) + "metadata.namespace": {ObjectMeta: kapi.ObjectMeta{Name: "requestName"}},
		string(fielderrors.ValidationErrorTypeRequired) + "metadata.name":      {ObjectMeta: kapi.ObjectMeta{Namespace: kapi.NamespaceDefault}},
	}

	for desc, tc := range testCases {
		errors := ValidateBuildRequest(tc)
		if len(desc) == 0 && len(errors) > 0 {
			t.Errorf("%s: Unexpected validation result: %v", desc, errors)
		}
		if len(desc) > 0 && len(errors) != 1 {
			t.Errorf("%s: Unexpected validation result: %v", desc, errors)
		}
		if len(desc) > 0 {
			err := errors[0].(*fielderrors.ValidationError)
			errDesc := string(err.Type) + err.Field
			if desc != errDesc {
				t.Errorf("Unexpected validation result for %s: expected %s, got %s", err.Field, desc, errDesc)
			}
		}
	}
}

func TestValidateSource(t *testing.T) {
	dockerfile := "FROM something"
	validGitURL := "https://github.com/some/server.git"
	errorCases := []struct {
		t        fielderrors.ValidationErrorType
		path     string
		source   *buildapi.BuildSource
		ok       bool
		multiple bool
	}{
		{
			t:    fielderrors.ValidationErrorTypeRequired,
			path: "git.uri",
			source: &buildapi.BuildSource{
				Type: buildapi.BuildSourceGit,
				Git: &buildapi.GitBuildSource{
					URI: "",
				},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "git.uri",
			source: &buildapi.BuildSource{
				Type: buildapi.BuildSourceGit,
				Git: &buildapi.GitBuildSource{
					URI: "::",
				},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "contextDir",
			source: &buildapi.BuildSource{
				Type:       buildapi.BuildSourceDockerfile,
				Dockerfile: &dockerfile,
				ContextDir: "../file",
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "git",
			source: &buildapi.BuildSource{
				Type:       buildapi.BuildSourceDockerfile,
				Dockerfile: &dockerfile,
				Git:        &buildapi.GitBuildSource{},
				Binary:     &buildapi.BinaryBuildSource{},
			},
			multiple: true,
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "binary",
			source: &buildapi.BuildSource{
				Type: buildapi.BuildSourceGit,
				Git: &buildapi.GitBuildSource{
					URI: validGitURL,
				},
				Binary: &buildapi.BinaryBuildSource{},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "git",
			source: &buildapi.BuildSource{
				Type:   buildapi.BuildSourceBinary,
				Git:    &buildapi.GitBuildSource{},
				Binary: &buildapi.BinaryBuildSource{},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeRequired,
			path: "binary",
			source: &buildapi.BuildSource{
				Type: buildapi.BuildSourceBinary,
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "binary.asFile",
			source: &buildapi.BuildSource{
				Type:   buildapi.BuildSourceBinary,
				Binary: &buildapi.BinaryBuildSource{AsFile: "/a/path"},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "binary.asFile",
			source: &buildapi.BuildSource{
				Type:   buildapi.BuildSourceBinary,
				Binary: &buildapi.BinaryBuildSource{AsFile: "/"},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "binary.asFile",
			source: &buildapi.BuildSource{
				Type:   buildapi.BuildSourceBinary,
				Binary: &buildapi.BinaryBuildSource{AsFile: "a\\b"},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "binary.asFile",
			source: &buildapi.BuildSource{
				Type:   buildapi.BuildSourceBinary,
				Binary: &buildapi.BinaryBuildSource{AsFile: "/././file"},
			},
			ok: true,
		},
		{
			t:    fielderrors.ValidationErrorTypeInvalid,
			path: "type",
			source: &buildapi.BuildSource{
				Type:   "invalidType",
				Binary: &buildapi.BinaryBuildSource{AsFile: "/././file"},
			},
		},
		{
			t:    fielderrors.ValidationErrorTypeRequired,
			path: "type",
			source: &buildapi.BuildSource{
				Binary: &buildapi.BinaryBuildSource{AsFile: "/././file"},
			},
		},
	}
	for i, tc := range errorCases {
		errors := validateSource(tc.source)
		switch len(errors) {
		case 0:
			if !tc.ok {
				t.Errorf("%d: Unexpected validation result: %v", i, errors)
			}
			continue
		case 1:
			if tc.ok || tc.multiple {
				t.Errorf("%d: Unexpected validation result: %v", i, errors)
				continue
			}
		default:
			if tc.ok || !tc.multiple {
				t.Errorf("%d: Unexpected validation result: %v", i, errors)
				continue
			}
		}
		err := errors[0].(*fielderrors.ValidationError)
		if err.Type != tc.t {
			t.Errorf("%d: Unexpected error type: %s", i, err.Type)
		}
		if err.Field != tc.path {
			t.Errorf("%d: Unexpected error path: %s", i, err.Field)
		}
	}
}

func TestValidateBuildSpec(t *testing.T) {
	zero := int64(0)
	longString := strings.Repeat("1234567890", 100*61)
	//shortString := "FROM foo"
	errorCases := []struct {
		err string
		*buildapi.BuildSpec
	}{
		// 0
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "output.to.name",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "some/long/value/with/no/meaning",
					},
				},
			},
		},
		// 1
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "output.to.kind",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "Foo",
						Name: "test",
					},
				},
			},
		},
		// 2
		{
			string(fielderrors.ValidationErrorTypeRequired) + "output.to.kind",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{},
				},
			},
		},
		// 3
		{
			string(fielderrors.ValidationErrorTypeRequired) + "output.to.name",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "ImageStreamTag",
					},
				},
			},
		},
		// 4
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "output.to.name",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind:      "ImageStreamTag",
						Name:      "missingtag",
						Namespace: "subdomain",
					},
				},
			},
		},
		// 5
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "output.to.namespace",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind:      "ImageStreamTag",
						Name:      "test:tag",
						Namespace: "not_a_valid_subdomain",
					},
				},
			},
		},
		// 6
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "strategy.type",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{Type: "invalid-type"},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 7
		{
			string(fielderrors.ValidationErrorTypeRequired) + "strategy.type",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 8
		// invalid because from is not specified in the
		// sti strategy definition
		{
			string(fielderrors.ValidationErrorTypeRequired) + "strategy.stiStrategy.from.kind",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.SourceBuildStrategyType,
					SourceStrategy: &buildapi.SourceBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 9
		// Invalid because from.name is not specified
		{
			string(fielderrors.ValidationErrorTypeRequired) + "strategy.stiStrategy.from.name",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type: buildapi.SourceBuildStrategyType,
					SourceStrategy: &buildapi.SourceBuildStrategy{
						From: kapi.ObjectReference{
							Kind: "DockerImage",
						},
					},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 10
		// invalid because from name is a bad format
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "strategy.stiStrategy.from.name",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type: buildapi.SourceBuildStrategyType,
					SourceStrategy: &buildapi.SourceBuildStrategy{
						From: kapi.ObjectReference{Kind: "ImageStreamTag", Name: "bad format"},
					},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 11
		// invalid because from is not specified in the
		// custom strategy definition
		{
			string(fielderrors.ValidationErrorTypeRequired) + "strategy.customStrategy.from.kind",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.CustomBuildStrategyType,
					CustomStrategy: &buildapi.CustomBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 12
		// invalid because from.name is not specified in the
		// custom strategy definition
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "strategy.customStrategy.from.name",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type: buildapi.CustomBuildStrategyType,
					CustomStrategy: &buildapi.CustomBuildStrategy{
						From: kapi.ObjectReference{Kind: "ImageStreamTag", Name: "bad format"},
					},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 13
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "source.dockerfile",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type:       buildapi.BuildSourceDockerfile,
					Dockerfile: &longString,
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
			},
		},
		// 14
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "source.dockerfile",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type:       buildapi.BuildSourceGit,
					Dockerfile: &longString,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
			},
		},
		// 15
		// invalid because CompletionDeadlineSeconds <= 0
		{
			string(fielderrors.ValidationErrorTypeInvalid) + "completionDeadlineSeconds",
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
					ContextDir: "context",
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
				CompletionDeadlineSeconds: &zero,
			},
		},
	}

	for count, config := range errorCases {
		errors := validateBuildSpec(config.BuildSpec)
		if len(errors) != 1 {
			t.Errorf("Test[%d] %s: Unexpected validation result: %v", count, config.err, errors)
			continue
		}
		err := errors[0].(*fielderrors.ValidationError)
		errDesc := string(err.Type) + err.Field
		if config.err != errDesc {
			t.Errorf("Test[%d] Unexpected validation result for %s: expected %s, got %s", count, err.Field, config.err, errDesc)
		}
	}
}

func TestValidateBuildSpecSuccess(t *testing.T) {
	shortString := "FROM foo"
	testCases := []struct {
		*buildapi.BuildSpec
	}{
		// 0
		{
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type: buildapi.SourceBuildStrategyType,
					SourceStrategy: &buildapi.SourceBuildStrategy{
						From: kapi.ObjectReference{
							Kind: "DockerImage",
							Name: "reponame",
						},
					},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 1
		{
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type: buildapi.CustomBuildStrategyType,
					CustomStrategy: &buildapi.CustomBuildStrategy{
						From: kapi.ObjectReference{
							Kind: "ImageStreamTag",
							Name: "imagestreamname:tag",
						},
					},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 2
		{
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type:           buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 3
		{
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type: buildapi.BuildSourceGit,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type: buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{
						From: &kapi.ObjectReference{
							Kind: "ImageStreamImage",
							Name: "imagestreamimage",
						},
					},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
		// 4
		{
			&buildapi.BuildSpec{
				Source: buildapi.BuildSource{
					Type:       buildapi.BuildSourceDockerfile,
					Dockerfile: &shortString,
					Git: &buildapi.GitBuildSource{
						URI: "http://github.com/my/repository",
					},
				},
				Strategy: buildapi.BuildStrategy{
					Type: buildapi.DockerBuildStrategyType,
					DockerStrategy: &buildapi.DockerBuildStrategy{
						From: &kapi.ObjectReference{
							Kind: "ImageStreamImage",
							Name: "imagestreamimage",
						},
					},
				},
				Output: buildapi.BuildOutput{
					To: &kapi.ObjectReference{
						Kind: "DockerImage",
						Name: "repository/data",
					},
				},
			},
		},
	}

	for count, config := range testCases {
		errors := validateBuildSpec(config.BuildSpec)
		if len(errors) != 0 {
			t.Errorf("Test[%d] Unexpected validation error: %v", count, errors)
		}
	}

}

func TestValidateTrigger(t *testing.T) {
	tests := map[string]struct {
		trigger  buildapi.BuildTriggerPolicy
		expected []*fielderrors.ValidationError
	}{
		"trigger without type": {
			trigger:  buildapi.BuildTriggerPolicy{},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("type")},
		},
		"trigger with unknown type": {
			trigger: buildapi.BuildTriggerPolicy{
				Type: "UnknownTriggerType",
			},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldInvalid("type", "", "")},
		},
		"GitHub type with no github webhook": {
			trigger:  buildapi.BuildTriggerPolicy{Type: buildapi.GitHubWebHookBuildTriggerType},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("github")},
		},
		"GitHub trigger with no secret": {
			trigger: buildapi.BuildTriggerPolicy{
				Type:          buildapi.GitHubWebHookBuildTriggerType,
				GitHubWebHook: &buildapi.WebHookTrigger{},
			},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("github.secret")},
		},
		"GitHub trigger with generic webhook": {
			trigger: buildapi.BuildTriggerPolicy{
				Type: buildapi.GitHubWebHookBuildTriggerType,
				GenericWebHook: &buildapi.WebHookTrigger{
					Secret: "secret101",
				},
			},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("github")},
		},
		"Generic trigger with no generic webhook": {
			trigger:  buildapi.BuildTriggerPolicy{Type: buildapi.GenericWebHookBuildTriggerType},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("generic")},
		},
		"Generic trigger with no secret": {
			trigger: buildapi.BuildTriggerPolicy{
				Type:           buildapi.GenericWebHookBuildTriggerType,
				GenericWebHook: &buildapi.WebHookTrigger{},
			},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("generic.secret")},
		},
		"Generic trigger with github webhook": {
			trigger: buildapi.BuildTriggerPolicy{
				Type: buildapi.GenericWebHookBuildTriggerType,
				GitHubWebHook: &buildapi.WebHookTrigger{
					Secret: "secret101",
				},
			},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("generic")},
		},
		"ImageChange trigger without params": {
			trigger: buildapi.BuildTriggerPolicy{
				Type: buildapi.ImageChangeBuildTriggerType,
			},
			expected: []*fielderrors.ValidationError{fielderrors.NewFieldRequired("imageChange")},
		},
		"valid GitHub trigger": {
			trigger: buildapi.BuildTriggerPolicy{
				Type: buildapi.GitHubWebHookBuildTriggerType,
				GitHubWebHook: &buildapi.WebHookTrigger{
					Secret: "secret101",
				},
			},
		},
		"valid Generic trigger": {
			trigger: buildapi.BuildTriggerPolicy{
				Type: buildapi.GenericWebHookBuildTriggerType,
				GenericWebHook: &buildapi.WebHookTrigger{
					Secret: "secret101",
				},
			},
		},
		"valid ImageChange trigger": {
			trigger: buildapi.BuildTriggerPolicy{
				Type: buildapi.ImageChangeBuildTriggerType,
				ImageChange: &buildapi.ImageChangeTrigger{
					LastTriggeredImageID: "asdf1234",
				},
			},
		},
		"valid ImageChange trigger with empty fields": {
			trigger: buildapi.BuildTriggerPolicy{
				Type:        buildapi.ImageChangeBuildTriggerType,
				ImageChange: &buildapi.ImageChangeTrigger{},
			},
		},
	}
	for desc, test := range tests {
		errors := validateTrigger(&test.trigger)
		if len(test.expected) == 0 {
			if len(errors) != 0 {
				t.Errorf("%s: Got unexpected validation errors: %#v", desc, errors)
			}
			continue
		}
		if len(errors) != 1 {
			t.Errorf("%s: Expected one validation error, got %d", desc, len(errors))
			for i, err := range errors {
				validationError := err.(*fielderrors.ValidationError)
				t.Errorf("  %d. %v", i+1, validationError)
			}
			continue
		}
		err := errors[0]
		validationError := err.(*fielderrors.ValidationError)
		if validationError.Type != test.expected[0].Type {
			t.Errorf("%s: Unexpected error type: %s", desc, validationError.Type)
		}
		if validationError.Field != test.expected[0].Field {
			t.Errorf("%s: Unexpected error field: %s", desc, validationError.Field)
		}
	}
}
