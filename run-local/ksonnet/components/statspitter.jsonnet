local env = std.extVar('__ksonnet/environments');
local params = std.extVar('__ksonnet/params').components.ss;
local k = import 'k.libsonnet';
local deployment = k.apps.v1beta1.deployment;
local container = k.apps.v1beta1.deployment.mixin.spec.template.spec.containersType;
local containerPort = container.portsType;
local service = k.core.v1.service;
local servicePort = k.core.v1.service.mixin.spec.portsType;

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
    .withImagePullPolicy('Never'),
    labels
  )
  .withTerminationGracePeriodSeconds(1)
  + deployment.mixin.spec.template.metadata.withAnnotations({
    'freshtracks.io.scrape': 'true',
  });

k.core.v1.list.new([appService, appDeployment])
