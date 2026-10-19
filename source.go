package sources

import (
	"context"
	"fmt"

	"github.com/aosanya/mwanachama-backend-shared/specstore"
)

func (m *sourceManager) CreateSource(ctx context.Context, in Source) (Source, error) {
	if err := m.checks(roleSource, in, ErrInvalidSource); err != nil {
		return Source{}, err
	}
	in.ID = newID()
	in.CreatedAt = m.now()
	in.UpdatedAt = in.CreatedAt

	if err := m.st.Insert(ctx, roleSource, in); err != nil {
		return Source{}, fmt.Errorf("CreateSource: %w", err)
	}
	return in, nil
}

func (m *sourceManager) GetSource(ctx context.Context, sourceID string) (Source, error) {
	var out Source
	q := m.st.Query(ctx, roleSource).Where("id = ?", sourceID)
	if err := m.st.Take(q, roleSource, &out, ErrSourceNotFound); err != nil {
		return Source{}, err
	}
	return out, nil
}

func (m *sourceManager) ListSources(ctx context.Context, filter SourceFilter) ([]Source, error) {
	q := m.st.Query(ctx, roleSource)
	if filter.Kind != "" {
		q = q.Where("kind = ?", filter.Kind)
	}
	out, err := specstore.List[Source](m.st, q.Order("created_at desc"), roleSource)
	if err != nil {
		return nil, fmt.Errorf("ListSources: %w", err)
	}
	return out, nil
}

func (m *sourceManager) UpdateSource(ctx context.Context, in Source) (Source, error) {
	held, err := m.GetSource(ctx, in.ID)
	if err != nil {
		return Source{}, err
	}

	held.Title = in.Title
	held.Kind = in.Kind
	held.Citation = in.Citation
	held.Locator = in.Locator
	held.ContentRef = in.ContentRef
	held.IssuedOn = in.IssuedOn
	held.Properties = in.Properties
	held.UpdatedAt = m.now()

	if err := m.checks(roleSource, held, ErrInvalidSource); err != nil {
		return Source{}, err
	}
	if err := m.update(ctx, roleSource, held.ID, held); err != nil {
		return Source{}, fmt.Errorf("UpdateSource: %w", err)
	}
	return held, nil
}
