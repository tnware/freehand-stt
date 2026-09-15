package storage

import (
	"context"
	"errors"
	"github.com/tnware/freehand-stt/internal/config"
	"github.com/tnware/freehand-stt/internal/storage/dbgen"
)

func writeInstances(ctx context.Context, q *dbgen.Queries, v config.Settings) error {
	old, e := q.ListManagedInstances(ctx)
	if e != nil {
		return e
	}
	for _, i := range v.ManagedRuntimes {
		for _, o := range old {
			if o.ID == i.ID && o.Provider != string(i.Provider) {
				return errors.New("runtime provider is immutable")
			}
		}
		if e = q.PutManagedInstance(ctx, dbgen.PutManagedInstanceParams{ID: i.ID, Name: i.Name, Provider: string(i.Provider), Model: i.Model, SpeechModel: i.SpeechModel, AutoStart: boolean(i.AutoStart)}); e != nil {
			return e
		}
	}
	return nil
}
func deleteInstances(ctx context.Context, q *dbgen.Queries, v config.Settings) error {
	old, e := q.ListManagedInstances(ctx)
	if e != nil {
		return e
	}
	for _, o := range old {
		found := false
		for _, i := range v.ManagedRuntimes {
			if i.ID == o.ID {
				found = true
			}
		}
		if !found {
			if e = q.DeleteManagedInstance(ctx, o.ID); e != nil {
				return e
			}
		}
	}
	return nil
}
