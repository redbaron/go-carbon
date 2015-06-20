package carbon

// Carbon - main application controller
type Carbon struct {
}

// New returns new instance of Carbon
func New() *Carbon {
	return &Carbon{}
}

// Configure init or change carbon configuration
func (carbon *Carbon) Configure(config *Config) {

}
