package storage

import (
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/managedruntime"
	"github.com/tnware/freehand-stt/internal/modelsettings"
	"github.com/tnware/freehand-stt/internal/savedconnection"
)

func validateRememberedModel(e modelsettings.Entry, d savedconnection.Details, instances []managedruntime.Instance) error {
	if d.ManagedInstanceID == "" {
		return modelsettings.Validate(e, d)
	}
	v := config.Default()
	v.ManagedRuntimes = append([]managedruntime.Instance{}, instances...)
	found := false
	for n := range v.ManagedRuntimes {
		if v.ManagedRuntimes[n].ID == d.ManagedInstanceID {
			v.ManagedRuntimes[n].Model = e.Model
			found = true
		}
	}
	if !found {
		return errors.New("remembered runtime is unavailable")
	}
	_, c, err := config.ManagedContract(v, d.ManagedInstanceID, savedconnection.Role(e.Purpose))
	if err != nil {
		return err
	}
	if e.Options.Profile != c.ModelProfile {
		return errors.New("remembered managed profile does not match catalog model")
	}
	v = modelsettings.Apply(savedconnection.Apply(v, e.Purpose, d), e.Purpose, e.Model, e.Options)
	if modelsettings.Extract(v, e.Purpose) != e.Options {
		return errors.New("remembered options belong to another task")
	}
	return config.Validate(v)
}
