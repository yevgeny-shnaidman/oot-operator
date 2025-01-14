package controllers

import (
	"context"
	"errors"
	"fmt"

	ootov1alpha1 "github.com/qbarrand/oot-operator/api/v1alpha1"
	"github.com/qbarrand/oot-operator/controllers/constants"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

//go:generate mockgen -source=daemonset.go -package=controllers -destination=mock_daemonset.go

type DaemonSetCreator interface {
	GarbageCollect(ctx context.Context, existingDS map[string]*appsv1.DaemonSet, validKernels sets.String) ([]string, error)
	ModuleDaemonSetsByKernelVersion(ctx context.Context, mod ootov1alpha1.Module) (map[string]*appsv1.DaemonSet, error)
	SetAsDesired(ds *appsv1.DaemonSet, image string, mod ootov1alpha1.Module, kernelVersion string) error
}

type daemonSetGenerator struct {
	client      client.Client
	kernelLabel string
	scheme      *runtime.Scheme
}

func NewDaemonSetCreator(client client.Client, kernelLabel string, scheme *runtime.Scheme) *daemonSetGenerator {
	return &daemonSetGenerator{
		client:      client,
		kernelLabel: kernelLabel,
		scheme:      scheme,
	}
}

func (dc *daemonSetGenerator) GarbageCollect(ctx context.Context, existingDS map[string]*appsv1.DaemonSet, validKernels sets.String) ([]string, error) {
	deleted := make([]string, 0)

	for kernelVersion, ds := range existingDS {
		if !validKernels.Has(kernelVersion) {
			if err := dc.client.Delete(ctx, ds); err != nil {
				return nil, fmt.Errorf("could not delete DaemonSet %s: %v", ds.Name, err)
			}

			deleted = append(deleted, ds.Name)
		}
	}

	return deleted, nil
}

func (dc *daemonSetGenerator) ModuleDaemonSetsByKernelVersion(ctx context.Context, mod ootov1alpha1.Module) (map[string]*appsv1.DaemonSet, error) {
	dsList := appsv1.DaemonSetList{}

	opts := []client.ListOption{
		client.MatchingLabels(map[string]string{constants.ModuleNameLabel: mod.Name}),
		client.InNamespace(mod.Namespace),
	}

	if err := dc.client.List(ctx, &dsList, opts...); err != nil {
		return nil, fmt.Errorf("could not list DaemonSets: %v", err)
	}

	dsByKernelVersion := make(map[string]*appsv1.DaemonSet, len(dsList.Items))

	for i := 0; i < len(dsList.Items); i++ {
		ds := dsList.Items[i]

		kernelVersion := ds.Labels[dc.kernelLabel]

		if dsByKernelVersion[kernelVersion] != nil {
			return nil, fmt.Errorf("multiple DaemonSets found for kernel %q", kernelVersion)
		}

		dsByKernelVersion[kernelVersion] = &ds
	}

	return dsByKernelVersion, nil
}

func (dc *daemonSetGenerator) SetAsDesired(ds *appsv1.DaemonSet, image string, mod ootov1alpha1.Module, kernelVersion string) error {
	if ds == nil {
		return errors.New("ds cannot be nil")
	}

	if image == "" {
		return errors.New("image cannot be empty")
	}

	if kernelVersion == "" {
		return errors.New("kernelVersion cannot be empty")
	}

	standardLabels := map[string]string{
		constants.ModuleNameLabel: mod.Name,
		dc.kernelLabel:            kernelVersion,
	}

	labels := ds.GetLabels()

	if labels == nil {
		labels = make(map[string]string, len(standardLabels))
	}

	for k, v := range standardLabels {
		labels[k] = v
	}

	ds.SetLabels(labels)

	nodeSelector := CopyMapStringString(mod.Spec.Selector)
	nodeSelector[dc.kernelLabel] = kernelVersion

	const (
		kubeletDevicePluginsVolumeName = "kubelet-device-plugins"
		kubeletDevicePluginsPath       = "/var/lib/kubelet/device-plugins"
		nodeLibModulesPath             = "/lib/modules"
		nodeLibModulesVolumeName       = "node-lib-modules"
		nodeUsrLibModulesPath          = "/usr/lib/modules"
		nodeUsrLibModulesVolumeName    = "node-usr-lib-modules"
	)

	containers := make([]v1.Container, 0, 2)

	driverContainerVolumeMounts := []v1.VolumeMount{
		{
			Name:      nodeLibModulesVolumeName,
			ReadOnly:  true,
			MountPath: nodeLibModulesPath,
		},
		{
			Name:      nodeUsrLibModulesVolumeName,
			ReadOnly:  true,
			MountPath: nodeUsrLibModulesPath,
		},
	}

	driverContainer := mod.Spec.DriverContainer
	driverContainer.Name = "driver-container"
	driverContainer.Image = image
	driverContainer.VolumeMounts = append(driverContainer.VolumeMounts, driverContainerVolumeMounts...)

	containers = append(containers, driverContainer)

	hostPathDirectory := v1.HostPathDirectory
	varTrue := true

	volumes := []v1.Volume{
		{
			Name: nodeLibModulesVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: nodeLibModulesPath,
					Type: &hostPathDirectory,
				},
			},
		},
		{
			Name: nodeUsrLibModulesVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: nodeUsrLibModulesPath,
					Type: &hostPathDirectory,
				},
			},
		},
	}

	if mod.Spec.DevicePlugin != nil {
		devicePlugin := *mod.Spec.DevicePlugin
		devicePlugin.Name = "device-plugin"

		if devicePlugin.SecurityContext == nil {
			devicePlugin.SecurityContext = &v1.SecurityContext{}
		}

		devicePlugin.SecurityContext.Privileged = &varTrue

		devicePluginsVolumeMount := v1.VolumeMount{
			Name:      kubeletDevicePluginsVolumeName,
			MountPath: kubeletDevicePluginsPath,
		}

		devicePlugin.VolumeMounts = append(devicePlugin.VolumeMounts, devicePluginsVolumeMount)

		containers = append(containers, devicePlugin)

		devicePluginsVolume := v1.Volume{
			Name: kubeletDevicePluginsVolumeName,
			VolumeSource: v1.VolumeSource{
				HostPath: &v1.HostPathVolumeSource{
					Path: kubeletDevicePluginsPath,
					Type: &hostPathDirectory,
				},
			},
		}

		volumes = append(volumes, devicePluginsVolume)
	}

	volumes = append(volumes, mod.Spec.AdditionalVolumes...)

	ds.Spec = appsv1.DaemonSetSpec{
		Selector: &metav1.LabelSelector{MatchLabels: standardLabels},
		Template: v1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{Labels: standardLabels},
			Spec: v1.PodSpec{
				NodeSelector:       nodeSelector,
				Containers:         containers,
				ServiceAccountName: mod.Spec.ServiceAccountName,
				Volumes:            volumes,
			},
		},
	}

	if err := controllerutil.SetControllerReference(&mod, ds, dc.scheme); err != nil {
		return fmt.Errorf("could not set the owner reference: %v", err)
	}

	return nil
}

// CopyMapStringString returns a deep copy of m.
func CopyMapStringString(m map[string]string) map[string]string {
	n := make(map[string]string, len(m))

	for k, v := range m {
		n[k] = v
	}

	return n
}
