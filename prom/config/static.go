package config

// Group is a set of targets with a common label set(production , test, staging etc.).
type TargetGroup struct {
	// Targets is a list of targets identified by a label set. Each target is
	// uniquely identifiable in the group by its address label.
	Targets []LabelSet
	// Labels is a set of labels that is common across all targets in the group.
	Labels LabelSet

	// Source is an identifier that describes a group of targets.
	Source string
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (tg *TargetGroup) UnmarshalYAML(unmarshal func(interface{}) error) error {
	g := struct {
		Targets []string `yaml:"targets"`
		Labels  LabelSet `yaml:"labels"`
	}{}
	if err := unmarshal(&g); err != nil {
		return err
	}
	tg.Targets = make([]LabelSet, 0, len(g.Targets))
	for _, t := range g.Targets {
		tg.Targets = append(tg.Targets, LabelSet{
			AddressLabel: LabelValue(t),
		})
	}
	tg.Labels = g.Labels
	return nil
}

// MarshalYAML implements the yaml.Marshaler interface.
func (tg TargetGroup) MarshalYAML() (interface{}, error) {
	g := &struct {
		Targets []string `yaml:"targets"`
		Labels  LabelSet `yaml:"labels,omitempty"`
	}{
		Targets: make([]string, 0, len(tg.Targets)),
		Labels:  tg.Labels,
	}
	for _, t := range tg.Targets {
		g.Targets = append(g.Targets, string(t[AddressLabel]))
	}
	return g, nil
}

const (
	AlertNameLabel      = "alertname"
	ExportedLabelPrefix = "exported_"
	MetricNameLabel     = "__name__"
	SchemeLabel         = "__scheme__"
	AddressLabel        = "__address__"
	MetricsPathLabel    = "__metrics_path__"
	ReservedLabelPrefix = "__"
	MetaLabelPrefix     = "__meta_"
	TmpLabelPrefix      = "__tmp_"
	ParamLabelPrefix    = "__param_"
	JobLabel            = "job"
	InstanceLabel       = "instance"
	BucketLabel         = "le"
	QuantileLabel       = "quantile"
)
