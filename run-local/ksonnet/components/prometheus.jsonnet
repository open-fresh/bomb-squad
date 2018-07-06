local env = std.extVar('__ksonnet/environments');
local params = std.extVar('__ksonnet/params').components.prometheus;
local k = import 'k.libsonnet';
local deployment = k.apps.v1beta1.deployment;
local container = k.apps.v1beta1.deployment.mixin.spec.template.spec.containersType;
local containerPort = container.portsType;
local service = k.core.v1.service;
local servicePort = k.core.v1.service.mixin.spec.portsType;
local configMap = k.core.v1.configMap;
local labels = { app: params.name };

local prometheusService =
  service.new(
    'prometheus',
    labels,
    servicePort.new(params.prometheusServicePort, params.prometheusTargetPort)
    .withNodePort(params.prometheusNodePort)
  )
  .withType('NodePort');

local cm =
  configMap.new(
    params.name
  )
  + configMap.withData(
    {
      'prometheus.yml': importstr 'prometheus/prometheus.yml',
      //'rules.yml': importstr 'prometheus/rules.yml',
    },
  );

local dataVolumeMount = {
  name: 'prom-data',
  mountPath: '/data',
  readOnly: false,
};

local appDeployment =
  deployment.new(
    params.name,
    params.replicas,
    [
      container
      .new('prometheus', params.promImage)
      .withArgs([
        '--config.file=/etc/config/prometheus.yml',
        '--storage.tsdb.path=/data',
        '--storage.tsdb.retention=30m',
        '--storage.tsdb.min-block-duration=15m',
        '--storage.tsdb.max-block-duration=30m',
        '--web.console.libraries=/etc/prometheus/console_libraries',
        '--web.console.templates=/etc/prometheus/consoles',
        '--web.enable-lifecycle',
        '--query.timeout=10m',
      ])
      .withPorts(containerPort.new(params.prometheusTargetPort))
      .withImagePullPolicy('IfNotPresent')
      .withVolumeMounts([
        {
          name: 'prom-cfg',
          mountPath: '/etc/config',
          readOnly: true,
        },
        dataVolumeMount,
      ]),
    ],
    labels
  )
  .withTerminationGracePeriodSeconds(1)
  + deployment.mixin.spec.template.spec.withVolumes([
    {
      name: 'prom-cfg',
      configMap: {
        name: params.name,
      },
    },
    {
      name: 'prom-data',
      emptyDir: {},
    },
  ]);

k.core.v1.list.new([prometheusService, appDeployment, cm])
