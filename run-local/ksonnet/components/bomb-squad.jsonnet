local env = std.extVar('__ksonnet/environments');
local params = std.extVar('__ksonnet/params').components['bomb-squad'];
local k = import 'k.libsonnet';
local deployment = k.apps.v1beta1.deployment;
local container = k.apps.v1beta1.deployment.mixin.spec.template.spec.containersType;
local containerPort = container.portsType;
local service = k.core.v1.service;
local servicePort = k.core.v1.service.mixin.spec.portsType;
local configMap = k.core.v1.configMap;
local labels = { app: params.name };

local bombSquadService =
  service.new(
    'bomb-squad',
    labels,
    servicePort.new(params.servicePort, params.containerPort)
    .withNodePort(params.nodePort)
  )
  .withType('NodePort');

local appDeployment =
  deployment.new(
    params.name,
    params.replicas,
    [
      container
      .new('bomb-squad', params.image + ':' + params.imageTag)
      .withArgs([
        '-prom-url=prometheus:9090',
      ])
      .withPorts(containerPort.new(params.containerPort))
      .withImagePullPolicy('Never'),
    ],
    labels
  )
  .withTerminationGracePeriodSeconds(1);
//  + deployment.mixin.spec.template.spec.withVolumes([
//    {
//      name: 'prom-cfg',
//      configMap: {
//        name: params.name,
//      },
//    },
//    ]);

k.core.v1.list.new([bombSquadService, appDeployment])
