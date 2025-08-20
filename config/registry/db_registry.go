package registry

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type DBServiceRegistry struct {
	DB *sql.DB
}

type DBEventRegistry struct {
	DB *sql.DB
}

func (r *DBServiceRegistry) RegisterService(ctx context.Context, svc ServiceRegistration) error {
	methods, err := json.Marshal(svc.Methods)
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO service_registry (service_name, methods, registered_at)
		VALUES ($1, $2, $3)
		ON CONFLICT (service_name) DO UPDATE SET methods = $2, registered_at = $3
	`, svc.ServiceName, methods, time.Now().UTC())
	return err
}

func (r *DBEventRegistry) RegisterEvent(ctx context.Context, evt EventRegistration) error {
	params, err := json.Marshal(evt.Parameters)
	if err != nil {
		return err
	}
	required, err := json.Marshal(evt.RequiredFields)
	if err != nil {
		return err
	}
	_, err = r.DB.ExecContext(ctx, `
		INSERT INTO event_registry (event_name, parameters, required_fields, registered_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (event_name) DO UPDATE SET parameters = $2, required_fields = $3, registered_at = $4
	`, evt.EventName, params, required, time.Now().UTC())
	return err
}

func (r *DBServiceRegistry) LoadAll(ctx context.Context) ([]ServiceRegistration, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT service_name, methods, registered_at FROM service_registry`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []ServiceRegistration
	for rows.Next() {
		var svc ServiceRegistration
		var methods []byte
		if err := rows.Scan(&svc.ServiceName, &methods, &svc.RegisteredAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(methods, &svc.Methods); err != nil {
			return nil, err
		}
		result = append(result, svc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *DBEventRegistry) LoadAll(ctx context.Context) ([]EventRegistration, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT event_name, parameters, required_fields, registered_at FROM event_registry`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []EventRegistration
	for rows.Next() {
		var evt EventRegistration
		var params, required []byte
		if err := rows.Scan(&evt.EventName, &params, &required, &evt.RegisteredAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(params, &evt.Parameters); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(required, &evt.RequiredFields); err != nil {
			return nil, err
		}
		result = append(result, evt)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}
