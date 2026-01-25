package generator

// GeneratorOption is a function that modifies a Generator
// It is used to configure the generator during initialization
// Must be safely ignored if the generator does not support it
type GeneratorOption = func(gen Generator)

// WithPackageName sets the package name for the generated Go code
func WithPackageName(name string) GeneratorOption {
	return func(gen Generator) {
		switch g := gen.(type) {
		case *GoGenerator:
			g.packageName = name
		}
	}
}
