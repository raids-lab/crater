package checkpoint

type Processor struct {
	Normalizer Normalizer
	Validator  Validator
}

func DefaultProcessor() Processor {
	return Processor{
		Normalizer: DefaultNormalizer(),
		Validator:  DefaultValidator(),
	}
}

func Prepare(input PrepareInput) (*Config, error) {
	return DefaultProcessor().Prepare(input)
}

func (p Processor) Prepare(input PrepareInput) (*Config, error) {
	if len(p.Normalizer.Policies.policies) == 0 {
		p.Normalizer = DefaultNormalizer()
	}
	if len(p.Validator.Rules) == 0 {
		p.Validator = DefaultValidator()
	}
	cfg, err := p.Normalizer.Normalize(input)
	if err != nil {
		return nil, err
	}
	if err := p.Validator.Validate(ValidationContext{
		Config:       cfg,
		VolumeMounts: input.VolumeMounts,
		Policies:     p.Normalizer.Policies,
	}); err != nil {
		return nil, err
	}
	return cfg, nil
}
