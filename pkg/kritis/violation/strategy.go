/*
Copyright 2018 Google LLC

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
package violation

import (
	"fmt"

	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"

	"github.com/grafeas/kritis/pkg/kritis/constants"
	"github.com/grafeas/kritis/pkg/kritis/pods"
	"github.com/grafeas/kritis/pkg/kritis/policy"
)

type Strategy interface {
	HandleViolation(image string, pod *v1.Pod, violations []policy.Violation) error
	HandleAttestation(image string, pod *v1.Pod, isAttested bool) error
}

type LoggingStrategy struct {
}

func (l *LoggingStrategy) HandleViolation(image string, pod *v1.Pod, violations []policy.Violation) error {
	glog.Info("handling violations via LoggingStrategy")
	if len(violations) == 0 {
		return nil
	}
	glog.Warningf("found violations in image %q", image)
	for _, v := range violations {
		glog.Warning(v.Reason())
	}
	return nil
}

func (l *LoggingStrategy) HandleAttestation(image string, pod *v1.Pod, isAttested bool) error {
	glog.Info("handling attestation via LoggingStrategy")
	if isAttested {
		glog.Infof("image %q has one or more valid attestation(s)", image)
	} else {
		glog.Infof("no valid attestations found for image %q, proceeding with next checks", image)
	}
	return nil
}

// AnnotationStrategy adds "InvalidImageSecPolicy" as a label and an annotation, with detailed info in the annotations
type AnnotationStrategy struct {
}

func (a *AnnotationStrategy) HandleViolation(image string, pod *v1.Pod, violations []policy.Violation) error {
	// First, remove "kritis.grafeas.io/invalidImageSecPolicy" label/annotation in case it doesn't apply anymore
	if err := pods.DeleteLabelsAndAnnotations(*pod, []string{constants.InvalidImageSecPolicy}, []string{constants.InvalidImageSecPolicy}); err != nil {
		return err
	}
	if len(violations) == 0 {
		return nil
	}
	// Now, construct labels and annotations
	labelValue := constants.InvalidImageSecPolicyLabelValue
	annotationValue := fmt.Sprintf("found %d CVEs", len(violations))
	for _, v := range violations {
		if v.Type() == policy.UnqualifiedImageViolation {
			annotationValue += fmt.Sprintf(", %s", v.Reason())
			break
		}
	}
	labels := map[string]string{constants.InvalidImageSecPolicy: labelValue}
	annotations := map[string]string{constants.InvalidImageSecPolicy: annotationValue}
	glog.Info(fmt.Sprintf("adding label %q and annotation %q", labelValue, annotationValue))
	return pods.AddLabelsAndAnnotations(*pod, labels, annotations)
}

func (a *AnnotationStrategy) HandleAttestation(image string, pod *v1.Pod, isAttested bool) error {
	// First, remove "kritis.grafeas.io/attestation" label/annotation in case it doesn't apply anymore
	if err := pods.DeleteLabelsAndAnnotations(*pod, []string{constants.ImageAttestation}, []string{constants.ImageAttestation}); err != nil {
		return err
	}
	lValue := constants.NoAttestationsLabelValue
	aValue := constants.NoAttestationsAnnotation
	if isAttested {
		lValue = constants.PreviouslyAttestedLabelValue
		aValue = constants.PreviouslyAttestedAnnotation
	}
	labels := map[string]string{constants.ImageAttestation: lValue}
	annotations := map[string]string{constants.ImageAttestation: aValue}
	glog.Info(fmt.Sprintf("adding label %q and annotation %q", lValue, aValue))
	return pods.AddLabelsAndAnnotations(*pod, labels, annotations)
}

// For unit testing.
type MemoryStrategy struct {
	Violations   map[string]bool
	Attestations map[string]bool
}

func (ms *MemoryStrategy) HandleViolation(image string, p *v1.Pod, v []policy.Violation) error {
	ms.Violations[image] = true
	return nil
}

func (ms *MemoryStrategy) HandleAttestation(image string, pod *v1.Pod, isAttested bool) error {
	ms.Attestations[image] = isAttested
	return nil
}
