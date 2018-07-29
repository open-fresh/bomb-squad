local env = std.extVar('__ksonnet/environments');
local params = std.extVar('__ksonnet/params').components.grafana;
local k = import 'k.libsonnet';
local deployment = k.apps.v1beta1.deployment;
local container = k.apps.v1beta1.deployment.mixin.spec.template.spec.containersType;
local containerPort = container.portsType;
local service = k.core.v1.service;
local servicePort = k.core.v1.service.mixin.spec.portsType;
local configMap = k.core.v1.configMap;

local targetPort = params.containerPort;
local labels = { app: params.name };

local appService =
  service.new(
    params.name,
    labels,
    servicePort.new(params.servicePort, targetPort)
    .withNodePort(params.nodePort)
  )
  .withType(params.type);

local appDeployment =
  deployment.new(
    params.name,
    params.replicas,
    container
    .new(params.name, params.image)
    .withPorts(containerPort.new(targetPort))
    .withImagePullPolicy('IfNotPresent')
    .withVolumeMounts([
      {
        name: 'ds',
        mountPath: '/etc/grafana/provisioning/datasources',
      },
      {
        name: 'db',
        mountPath: '/etc/grafana/provisioning/dashboards',
      },
      {
        name: 'prom2db',
        mountPath: '/var/lib/grafana/dashboards',
      },
    ]),
    labels
  )
  .withTerminationGracePeriodSeconds(1)
  + deployment.mixin.spec.template.metadata.withAnnotations({
    'freshtracks.io.scrape': 'true',
  })
  + deployment.mixin.spec.template.spec.withVolumes([
    {
      name: 'ds',
      configMap: {
        name: params.name,
        items: [{
          key: 'ds.yaml',
          path: 'ds.yaml',
        }],
      },
    },
    {
      name: 'db',
      configMap: {
        name: params.name,
        items: [{
          key: 'db.yaml',
          path: 'db.yaml',
        }],
      },
    },
    {
      name: 'prom2db',
      configMap: {
        name: params.name,
        items: [{
          key: 'prom2db.json',
          path: 'prom2db.json',
        }],
      },
    },
  ])
;

local appConfigMap =
  configMap.new(
    params.name
  )
  + configMap.withData(
    {
      'ds.yaml': importstr 'grafana/ds_provision.yaml',
      'db.yaml': importstr 'grafana/db_provision.yaml',
      'prom2db.json': importstr 'grafana/prom2db.json',
    },
  );

k.core.v1.list.new([appService, appDeployment, appConfigMap])
