package utils

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/inject"
	"github.com/wgliang/logcool/utils/logevent"
)

// Output base type interface.
type TypeOutputConfig interface {
	TypeConfig
	Event(event logevent.LogEvent) (err error)
}

// Output base type struct.
type OutputConfig struct {
	CommonConfig
}

// OutputHandler type interface.
type OutputHandler interface{}

var (
	mapOutputHandler = map[string]OutputHandler{}
)

// Registe OutputHandler.
func RegistOutputHandler(name string, handler OutputHandler) {
	mapOutputHandler[name] = handler
}

// Run Outputs.
func (t *Config) RunOutputs() (err error) {
	_, err = t.Injector.Invoke(t.runOutputs)
	return
}

// run Outputs.
func (t *Config) runOutputs(outchan OutChan, logger *logrus.Logger) (err error) {
	outputs, err := t.getOutputs()
	if err != nil {
		return
	}
	go func() {
		for {
			select {
			case event := <-outchan:
				for _, output := range outputs {
					go func(o TypeOutputConfig, e logevent.LogEvent) {
						if err = o.Event(e); err != nil {
							logger.Errorf("output failed: %v\n", err)
						}
					}(output, event)
				}
			}
		}
	}()
	return
}

// get Outputs.
func (t *Config) getOutputs() (outputs []TypeOutputConfig, err error) {
	for _, confraw := range t.OutputRaw {
		handler, ok := mapOutputHandler[confraw["type"].(string)]
		if !ok {
			err = errors.New(confraw["type"].(string))
			return
		}

		inj := inject.New()
		inj.SetParent(t)
		inj.Map(&confraw)

		refvs, err := inj.Invoke(handler)
		if err != nil {
			return []TypeOutputConfig{}, err
		}

		for _, refv := range refvs {
			if !refv.CanInterface() {
				continue
			}
			if conf, ok := refv.Interface().(TypeOutputConfig); ok {
				conf.SetInjector(inj)
				outputs = append(outputs, conf)
			}
		}
	}
	return
}
