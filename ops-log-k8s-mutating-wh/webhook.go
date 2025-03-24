// SPDX-FileCopyrightText: 2025 SAP SE or an SAP affiliate company and prysm contributors
//
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/klog/v2"
)

// Define the sidecar container
var sidecarContainer = corev1.Container{
	Name:  "prysm-sidecar",
	Image: os.Getenv("SIDECAR_IMAGE"),
	Args: []string{
		"local-producer",
		"ops-log",
		"--log-file=/var/log/ceph/ops-log.log",
		"--max-log-file-size=10",
		"--prometheus=true",
		"--prometheus-port=9090",
		"-v=info",
	},
	Ports: []corev1.ContainerPort{
		{
			Name:          "metrics",
			ContainerPort: 9090,
			Protocol:      corev1.ProtocolTCP,
		},
	},
	VolumeMounts: []corev1.VolumeMount{
		{Name: "rook-config-override", ReadOnly: true, MountPath: "/etc/ceph"},
		{Name: "ceph-daemons-sock-dir", MountPath: "/run/ceph"},
		{Name: "rook-ceph-log", MountPath: "/var/log/ceph"},
		{Name: "rook-ceph-crash", MountPath: "/var/lib/ceph/crash"},
	},
	Env: []corev1.EnvVar{
		{
			Name: "POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	},
}

// Check if Deployment belongs to RADOSGW (based on labels)
func isRadosgwDeployment(deployment *appsv1.Deployment) bool {
	labels := deployment.Labels
	return labels["app"] == "rook-ceph-rgw" &&
		labels["app.kubernetes.io/component"] == "cephobjectstores.ceph.rook.io" &&
		labels["app.kubernetes.io/created-by"] == "rook-ceph-operator" &&
		labels["app.kubernetes.io/managed-by"] == "rook-ceph-operator" &&
		labels["prysm-sidecar"] == "yes"
}

// Mutate deployments to add a sidecar (only for RADOSGW)
func mutateDeployment(req *admissionv1.AdmissionRequest) *admissionv1.AdmissionResponse {
	if req.Kind.Kind != "Deployment" {
		return &admissionv1.AdmissionResponse{Allowed: true, UID: req.UID}
	}

	// Deserialize the Deployment object
	deployment := appsv1.Deployment{}
	if err := json.Unmarshal(req.Object.Raw, &deployment); err != nil {
		klog.Errorf("Failed to unmarshal Deployment: %v", err)
		return &admissionv1.AdmissionResponse{Allowed: false, UID: req.UID}
	}

	// Skip mutation if not a RADOSGW deployment
	if !isRadosgwDeployment(&deployment) {
		return &admissionv1.AdmissionResponse{Allowed: true, UID: req.UID}
	}

	klog.Infof("Mutating deployment: %s", deployment.Name)

	// Check for annotation with env secret name
	annotations := deployment.Spec.Template.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}

	if secretName, ok := annotations["prysm-sidecar/sidecar-env-secret"]; ok && secretName != "" {
		klog.Infof("Injecting envFrom using secret: %s", secretName)
		sidecarContainer.EnvFrom = append(sidecarContainer.EnvFrom, corev1.EnvFromSource{
			SecretRef: &corev1.SecretEnvSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: secretName},
				Optional:             pointerTo(true),
			},
		})
	}

	// Find if the sidecar already exists
	sidecarIndex := -1
	for i, container := range deployment.Spec.Template.Spec.Containers {
		if container.Name == sidecarContainer.Name {
			sidecarIndex = i
			break
		}
	}

	var patches []map[string]any

	if sidecarIndex >= 0 {
		// Replace existing sidecar
		klog.Infof("Replacing existing sidecar container in deployment: %s", deployment.Name)
		patches = append(patches, map[string]any{
			"op":    "replace",
			"path":  fmt.Sprintf("/spec/template/spec/containers/%d", sidecarIndex),
			"value": sidecarContainer,
		})
	} else {
		// Add the sidecar if it does not exist
		klog.Infof("Adding new sidecar container in deployment: %s", deployment.Name)
		patches = append(patches, map[string]any{
			"op":    "add",
			"path":  "/spec/template/spec/containers/-",
			"value": sidecarContainer,
		})
	}

	// Ensure the required volumes are added
	volumeMap := make(map[string]corev1.Volume)
	for _, v := range deployment.Spec.Template.Spec.Volumes {
		volumeMap[v.Name] = v
	}

	// Marshal JSON patch
	patchBytes, err := json.Marshal(patches)
	if err != nil {
		klog.Errorf("Failed to marshal JSON patch: %v", err)
		return &admissionv1.AdmissionResponse{Allowed: false, UID: req.UID}
	}

	return &admissionv1.AdmissionResponse{
		Allowed:   true,
		UID:       req.UID,
		Patch:     patchBytes,
		PatchType: func() *admissionv1.PatchType { pt := admissionv1.PatchTypeJSONPatch; return &pt }(),
	}
}

// Handle admission requests
func mutateHandler(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusInternalServerError)
		return
	}

	ar := admissionv1.AdmissionReview{}
	if err := json.Unmarshal(body, &ar); err != nil {
		http.Error(w, "Failed to decode request body", http.StatusBadRequest)
		return
	}

	// Mutate if necessary
	resp := mutateDeployment(ar.Request)

	// Wrap response in AdmissionReview
	response := admissionv1.AdmissionReview{
		TypeMeta: ar.TypeMeta,
		Response: resp,
	}

	responseBytes, err := json.Marshal(response)
	if err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(responseBytes)
}

func pointerTo[T any](v T) *T {
	return &v
}
