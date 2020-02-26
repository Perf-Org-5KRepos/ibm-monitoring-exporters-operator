package model

import (
	"fmt"

	monitoringv1alpha1 "github.com/IBM/ibm-monitoring-exporters-operator/pkg/apis/monitoring/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

//NodeExporterService creates a Service object for node exporter
func NodeExporterService(cr *monitoringv1alpha1.Exporter) *v1.Service {
	return &v1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetNodeExporterObjName(cr),
			Namespace:   cr.Namespace,
			Labels:      getNodeExporterLabels(cr),
			Annotations: getNodeExporterAnnotations(cr),
		},
		Spec: v1.ServiceSpec{
			Ports:    getNodeExporterPorts(cr),
			Selector: getNodeExporterLabels(cr),
			Type:     "ClusterIP",
		},
	}
}

//UpdatedNodeExporterService creates updated Service object for node exporter
func UpdatedNodeExporterService(cr *monitoringv1alpha1.Exporter, currService *v1.Service) *v1.Service {
	newService := currService.DeepCopy()
	newService.ObjectMeta.Labels = getNodeExporterLabels(cr)
	newService.Spec.Ports = getNodeExporterPorts(cr)
	newService.Spec.Selector = getNodeExporterLabels(cr)
	return newService

}

//NodeExporterDaemonset creates a DaemonSet Object for node exporter
func NodeExporterDaemonset(cr *monitoringv1alpha1.Exporter) *appsv1.DaemonSet {
	containers := []v1.Container{*getNodeExporterContainer(cr), *getRouterContainer(cr, NODE)}
	daemonset := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      GetNodeExporterObjName(cr),
			Namespace: cr.Namespace,
			Labels:    getNodeExporterLabels(cr),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: getNodeExporterLabels(cr),
			},
			UpdateStrategy: appsv1.DaemonSetUpdateStrategy{
				Type: appsv1.RollingUpdateDaemonSetStrategyType,
			},
			MinReadySeconds: 5,
			Template: v1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:   GetNodeExporterObjName(cr),
					Labels: getNodeExporterLabels(cr),
					//TODO: it requires special privelege
					//Annotations: map[string]string{"scheduler.alpha.kubernetes.io/critical-pod": ""},
				},
				Spec: v1.PodSpec{
					//TODO: it requires special privelege
					//PriorityClassName: "system-cluster-critical",
					HostPID:     true,
					HostIPC:     false,
					HostNetwork: true,
					Containers:  containers,
					Volumes:     getVolumes(cr, NODE),
				},
			},
		},
	}
	if cr.Spec.ImagePullSecrets != nil && len(cr.Spec.ImagePullSecrets) != 0 {
		var secrets []v1.LocalObjectReference
		for _, secret := range cr.Spec.ImagePullSecrets {
			secrets = append(secrets, v1.LocalObjectReference{Name: secret})
		}
		daemonset.Spec.Template.Spec.ImagePullSecrets = secrets

	}
	if len(cr.Spec.NodeExporter.ServiceAccount) != 0 {
		daemonset.Spec.Template.Spec.ServiceAccountName = cr.Spec.NodeExporter.ServiceAccount
	}
	return daemonset

}

//UpdatedNodeExporterDeamonset update DaemonSet Object for node exporter
func UpdatedNodeExporterDeamonset(cr *monitoringv1alpha1.Exporter, currDaemonset *appsv1.DaemonSet) *appsv1.DaemonSet {
	newDaemonset := currDaemonset.DeepCopy()
	containers := []v1.Container{*getNodeExporterContainer(cr), *getRouterContainer(cr, NODE)}
	newDaemonset.ObjectMeta.Labels = getNodeExporterLabels(cr)
	newDaemonset.Spec.Selector = &metav1.LabelSelector{
		MatchLabels: getNodeExporterLabels(cr),
	}
	newDaemonset.Spec.Template.ObjectMeta.Labels = getNodeExporterLabels(cr)
	newDaemonset.Spec.Template.Spec.Containers = containers
	newDaemonset.Spec.Template.Spec.Volumes = getVolumes(cr, NODE)
	if cr.Spec.ImagePullSecrets != nil && len(cr.Spec.ImagePullSecrets) != 0 {
		var secrets []v1.LocalObjectReference
		for _, secret := range cr.Spec.ImagePullSecrets {
			secrets = append(secrets, v1.LocalObjectReference{Name: secret})
		}
		newDaemonset.Spec.Template.Spec.ImagePullSecrets = secrets

	}
	if len(cr.Spec.NodeExporter.ServiceAccount) != 0 {
		newDaemonset.Spec.Template.Spec.ServiceAccountName = cr.Spec.NodeExporter.ServiceAccount
	}
	return newDaemonset

}
func getNodeExporterContainer(cr *monitoringv1alpha1.Exporter) *v1.Container {
	drops := []v1.Capability{"ALL"}
	pe := false
	p := false
	rofs := true
	userID := int64(65534)
	noRoot := true
	container := &v1.Container{
		Name:            "nodeexporter",
		Image:           cr.Spec.NodeExporter.Image,
		ImagePullPolicy: cr.Spec.ImagePolicy,
		Resources:       cr.Spec.NodeExporter.Resource,
		SecurityContext: &v1.SecurityContext{
			RunAsUser:                &userID,
			RunAsNonRoot:             &noRoot,
			AllowPrivilegeEscalation: &pe,
			Privileged:               &p,
			ReadOnlyRootFilesystem:   &rofs,
			Capabilities: &v1.Capabilities{
				Drop: drops,
			},
		},
		Args: []string{"--path.procfs=/host/proc", "--path.sysfs=/host/sys", "--web.listen-address=127.0.0.1:" + fmt.Sprint(cr.Spec.NodeExporter.HostPort)},
		VolumeMounts: []v1.VolumeMount{
			{
				Name:      "proc",
				MountPath: "/host/proc",
				ReadOnly:  true,
			},
			{
				Name:      "sys",
				MountPath: "/host/sys",
				ReadOnly:  true,
			},
		},
	}

	return container
}

//GetNodeExporterObjName return name of node exporter service and daemonset
func GetNodeExporterObjName(cr *monitoringv1alpha1.Exporter) string {
	return cr.Name + "-nodeexporter"
}

func getNodeExporterLabels(cr *monitoringv1alpha1.Exporter) map[string]string {
	lables := make(map[string]string)
	lables["app"] = "ibm-monitoring"
	lables["component"] = "nodeexporter"
	for key, v := range cr.Labels {
		lables[key] = v
	}
	return lables
}

func getNodeExporterAnnotations(cr *monitoringv1alpha1.Exporter) map[string]string {
	annotations := make(map[string]string)
	annotations["prometheus.io/scrape"] = "true"
	annotations["prometheus.io/scheme"] = "https"
	annotations["skip.verify"] = "true"
	return annotations
}
func getNodeExporterPorts(cr *monitoringv1alpha1.Exporter) []v1.ServicePort {
	return []v1.ServicePort{
		v1.ServicePort{
			Name:       "metrics",
			Port:       cr.Spec.NodeExporter.ServicePort,
			TargetPort: intstr.IntOrString{Type: intstr.Int, IntVal: cr.Spec.NodeExporter.ServicePort},
			Protocol:   "TCP",
		},
	}

}