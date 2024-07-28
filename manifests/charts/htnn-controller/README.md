# htnn-controller

![Version: 0.1.5](https://img.shields.io/badge/Version-0.1.5-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 0.2.1](https://img.shields.io/badge/AppVersion-0.2.1-informational?style=flat-square)

A Helm chart for HTNN controller

## Install

To install the chart with the release `htnn-controller`:

```shell
helm repo add htnn https://mosn.github.io/htnn
helm repo update

helm install htnn-controller htnn/htnn-controller --namespace istio-system --create-namespace
```

For more information like how to configure and troubleshoot, please refer to the [Installation Guide](https://github.com/mosn/htnn/blob/main/site/content/en/docs/getting-started/installation.md).

## Uninstall

To uninstall the Helm release `htnn-controller`:

```shell
helm uninstall htnn-controller -n istio-system
```

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| spacewander |  |  |

## Requirements

| Repository | Name | Version |
|------------|------|---------|
| https://istio-release.storage.googleapis.com/charts | istio-base(base) | 1.21.3 |

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| base.enableIstioConfigCRDs | bool | `true` |  |
| experimental.stableValidationPolicy | bool | `false` |  |
| gateways.securityContext | object | `{}` |  |
| global.autoscalingv2API | bool | `true` |  |
| global.caAddress | string | `""` |  |
| global.caName | string | `""` |  |
| global.certSigners | list | `[]` |  |
| global.configCluster | bool | `false` |  |
| global.configValidation | bool | `true` |  |
| global.defaultPodDisruptionBudget.enabled | bool | `true` |  |
| global.defaultResources.requests.cpu | string | `"10m"` |  |
| global.externalIstiod | bool | `false` |  |
| global.hub | string | `"ghcr.io/mosn"` |  |
| global.imagePullPolicy | string | `""` |  |
| global.imagePullSecrets | list | `[]` |  |
| global.istioNamespace | string | `"istio-system"` |  |
| global.istiod.enableAnalysis | bool | `false` |  |
| global.jwtPolicy | string | `"third-party-jwt"` |  |
| global.logAsJson | bool | `false` |  |
| global.logging.level | string | `"default:info"` |  |
| global.meshID | string | `""` |  |
| global.meshNetworks | object | `{}` |  |
| global.mountMtlsCerts | bool | `false` |  |
| global.multiCluster.clusterName | string | `""` |  |
| global.multiCluster.enabled | bool | `false` |  |
| global.network | string | `""` |  |
| global.omitSidecarInjectorConfigMap | bool | `false` |  |
| global.operatorManageWebhooks | bool | `false` |  |
| global.pilotCertProvider | string | `"istiod"` |  |
| global.priorityClassName | string | `""` |  |
| global.proxy.autoInject | string | `"enabled"` |  |
| global.proxy.clusterDomain | string | `"cluster.local"` |  |
| global.proxy.componentLogLevel | string | `"misc:error"` |  |
| global.proxy.enableCoreDump | bool | `false` |  |
| global.proxy.excludeIPRanges | string | `""` |  |
| global.proxy.excludeInboundPorts | string | `""` |  |
| global.proxy.excludeOutboundPorts | string | `""` |  |
| global.proxy.image | string | `"htnn-proxy"` |  |
| global.proxy.includeIPRanges | string | `"*"` |  |
| global.proxy.includeInboundPorts | string | `"*"` |  |
| global.proxy.includeOutboundPorts | string | `""` |  |
| global.proxy.logLevel | string | `"warning"` |  |
| global.proxy.outlierLogPath | string | `""` |  |
| global.proxy.privileged | bool | `false` |  |
| global.proxy.readinessFailureThreshold | int | `4` |  |
| global.proxy.readinessInitialDelaySeconds | int | `0` |  |
| global.proxy.readinessPeriodSeconds | int | `15` |  |
| global.proxy.resources.limits.cpu | string | `"2000m"` |  |
| global.proxy.resources.limits.memory | string | `"1024Mi"` |  |
| global.proxy.resources.requests.cpu | string | `"100m"` |  |
| global.proxy.resources.requests.memory | string | `"128Mi"` |  |
| global.proxy.startupProbe.enabled | bool | `true` |  |
| global.proxy.startupProbe.failureThreshold | int | `600` |  |
| global.proxy.statusPort | int | `15020` |  |
| global.proxy.tracer | string | `"none"` |  |
| global.proxy_init.image | string | `"proxyv2"` |  |
| global.remotePilotAddress | string | `""` |  |
| global.sds.token.aud | string | `"istio-ca"` |  |
| global.sts.servicePort | int | `0` |  |
| global.tag | string | `"dev"` |  |
| istio_cni.chained | bool | `true` |  |
| istio_cni.provider | string | `"default"` |  |
| istiodRemote.injectionCABundle | string | `""` |  |
| istiodRemote.injectionPath | string | `"/inject"` |  |
| istiodRemote.injectionURL | string | `""` |  |
| meshConfig.defaultConfig.proxyMetadata.ISTIO_DELTA_XDS | string | `"true"` |  |
| meshConfig.enablePrometheusMerge | bool | `true` |  |
| ownerName | string | `""` |  |
| pilot.affinity | object | `{}` |  |
| pilot.autoscaleBehavior | object | `{}` |  |
| pilot.autoscaleEnabled | bool | `true` |  |
| pilot.autoscaleMax | int | `5` |  |
| pilot.autoscaleMin | int | `1` |  |
| pilot.cni.enabled | bool | `false` |  |
| pilot.cni.provider | string | `"default"` |  |
| pilot.configMap | bool | `true` |  |
| pilot.configSource.subscribedResources | list | `[]` |  |
| pilot.cpu.targetAverageUtilization | int | `80` |  |
| pilot.deploymentLabels | object | `{}` |  |
| pilot.env.HTNN_ENABLE_LDS_PLUGIN_VIA_ECDS | string | `"false"` |  |
| pilot.env.PILOT_ENABLE_HTNN | string | `"true"` |  |
| pilot.env.PILOT_ENABLE_HTNN_STATUS | string | `"true"` |  |
| pilot.env.PILOT_SCOPE_GATEWAY_TO_NAMESPACE | string | `"true"` |  |
| pilot.extraContainerArgs | list | `[]` |  |
| pilot.hub | string | `""` |  |
| pilot.image | string | `"htnn-controller"` |  |
| pilot.ipFamilies | list | `[]` |  |
| pilot.ipFamilyPolicy | string | `""` |  |
| pilot.jwksResolverExtraRootCA | string | `""` |  |
| pilot.keepaliveMaxServerConnectionAge | string | `"30m"` |  |
| pilot.memory | object | `{}` |  |
| pilot.nodeSelector | object | `{}` |  |
| pilot.podAnnotations | object | `{}` |  |
| pilot.podLabels | object | `{}` |  |
| pilot.replicaCount | int | `1` |  |
| pilot.resources.requests.cpu | string | `"500m"` |  |
| pilot.resources.requests.memory | string | `"2048Mi"` |  |
| pilot.rollingMaxSurge | string | `"100%"` |  |
| pilot.rollingMaxUnavailable | string | `"25%"` |  |
| pilot.seccompProfile | object | `{}` |  |
| pilot.serviceAccountAnnotations | object | `{}` |  |
| pilot.serviceAnnotations | object | `{}` |  |
| pilot.tag | string | `""` |  |
| pilot.taint.enabled | bool | `false` |  |
| pilot.taint.namespace | string | `""` |  |
| pilot.tolerations | list | `[]` |  |
| pilot.topologySpreadConstraints | list | `[]` |  |
| pilot.traceSampling | float | `1` |  |
| pilot.trustedZtunnelNamespace | string | `""` |  |
| pilot.volumeMounts | list | `[]` |  |
| pilot.volumes | list | `[]` |  |
| revision | string | `""` |  |
| revisionTags | list | `[]` |  |
| sidecarInjectorWebhook.alwaysInjectSelector | list | `[]` |  |
| sidecarInjectorWebhook.defaultTemplates | list | `[]` |  |
| sidecarInjectorWebhook.enableNamespacesByDefault | bool | `false` |  |
| sidecarInjectorWebhook.injectedAnnotations | object | `{}` |  |
| sidecarInjectorWebhook.neverInjectSelector | list | `[]` |  |
| sidecarInjectorWebhook.reinvocationPolicy | string | `"Never"` |  |
| sidecarInjectorWebhook.rewriteAppHTTPProbe | bool | `true` |  |
| sidecarInjectorWebhook.templates | object | `{}` |  |
| telemetry.enabled | bool | `true` |  |
| telemetry.v2.enabled | bool | `true` |  |
| telemetry.v2.prometheus.enabled | bool | `true` |  |
| telemetry.v2.stackdriver.enabled | bool | `false` |  |

