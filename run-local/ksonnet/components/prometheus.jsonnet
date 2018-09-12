local env = std.extVar('__ksonnet/environments');
local params = std.extVar('__ksonnet/params').components.prometheus;
local bs = std.extVar('__ksonnet/params').components['bomb-squad'];
local k = import 'k.libsonnet';
local deployment = k.apps.v1beta1.deployment;
local container = k.apps.v1beta1.deployment.mixin.spec.template.spec.containersType;
local containerPort = container.portsType;
local service = k.core.v1.service;
local servicePort = k.core.v1.service.mixin.spec.portsType;
local configMap = k.core.v1.configMap;
local labels = { app: params.name, sidecar: 'bomb-squad' };

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
    },
  );

local dataVolumeMount = {
  name: 'prom-data',
  mountPath: '/data',
  readOnly: false,
};

local serviceAccount =
  {
    apiVersion: 'v1',
    kind: 'ServiceAccount',
    metadata: {
      name: 'prometheus',
      namespace: 'default',
    },
  };

local clusterRoleBinding =
  {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'ClusterRoleBinding',
    metadata: {
      name: 'prometheus',
    },
    roleRef: {
      apiGroup: 'rbac.authorization.k8s.io',
      kind: 'ClusterRole',
      name: 'prometheus',
    },
    subjects: [
      {
        kind: 'ServiceAccount',
        name: 'prometheus',
        namespace: 'default',
      },
    ],
  };

local clusterRole =
  {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'ClusterRole',
    metadata: {
      name: 'prometheus',
    },
    rules: [
      {
        apiGroups: [
          '',
        ],
        resources: [
          'configmaps',
        ],
        verbs: [
          'get',
          'list',
          'watch',
          'patch',
          'update',
          'create',
        ],
      },
      {
        apiGroups: [
          '',
        ],
        resources: [
          'nodes',
          'nodes/proxy',
          'services',
          'endpoints',
          'resourcequotas',
          'replicationcontrollers',
          'limitranges',
          'pods',
          'namespaces',
          'persistentvolumeclaims',
        ],
        verbs: [
          'get',
          'list',
          'watch',
        ],
      },
      {
        apiGroups: [
          'extensions',
        ],
        resources: [
          'ingresses',
          'daemonsets',
          'daemonsets/status',
          'daemonsets/finalizers',
          'deployments',
          'replicasets',
        ],
        verbs: [
          'get',
          'list',
          'watch',
        ],
      },
      {
        apiGroups: [
          'batch',
        ],
        resources: [
          'cronjobs',
          'jobs',
        ],
        verbs: [
          'get',
          'list',
          'watch',
        ],
      },
      {
        apiGroups: [
          'apps',
        ],
        resources: [
          'statefulsets',
        ],
        verbs: [
          'get',
          'list',
          'watch',
        ],
      },
      {
        nonResourceURLs: [
          '/metrics',
        ],
        verbs: [
          'get',
        ],
      },
    ],
  }

;

local bombSquadContainer =
  container
  .new('bomb-squad', bs.image + ':' + bs.imageTag)
  .withPorts(containerPort.new(bs.containerPort))
  .withImagePullPolicy('Never')
  .withVolumeMounts([
    {
      name: 'bomb-squad-rules',
      mountPath: '/etc/config/bomb-squad',
      readOnly: false,
    },
  ]);

local appDeployment =
  deployment.new(
    params.name,
    params.replicas,
    [
      bombSquadContainer,
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
        {
          name: 'bomb-squad-rules',
          mountPath: '/etc/config/bomb-squad',
          readOnly: false,
        },
        dataVolumeMount,
      ]),
      container
      .new('config-reload', 'jimmidyson/configmap-reload:v0.1')
      .withArgs([
        '--volume-dir=/etc/config',
        '--webhook-url=http://localhost:9090/-/reload',
      ])
      .withImagePullPolicy('IfNotPresent')
      .withVolumeMounts([
        {
          name: 'prom-cfg',
          mountPath: '/etc/config',
          readOnly: true,
        },
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
    {
      name: 'bomb-squad-rules',
      emptyDir: {},
    },
  ])
  + deployment.mixin.spec.template.spec.withServiceAccount('prometheus');

k.core.v1.list.new([prometheusService, appDeployment, cm, serviceAccount, clusterRole, clusterRoleBinding])
