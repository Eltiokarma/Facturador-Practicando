package tenant

import (
	"context"
	"errors"
	"sync"

	"github.com/google/uuid"
)

// Manager cachea los Record en memoria para evitar ir a DB en cada
// emisión. Thread-safe.
type Manager struct {
	mu    sync.RWMutex
	items map[uuid.UUID]*Record
	store *Store
}

func NewManager(s *Store) *Manager {
	return &Manager{items: make(map[uuid.UUID]*Record), store: s}
}

// LoadAll carga todos los tenants de DB al caché. Llamar al arrancar.
func (m *Manager) LoadAll(ctx context.Context) error {
	list, err := m.store.ListAll(ctx)
	if err != nil {
		return err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, r := range list {
		m.items[r.ID] = r
	}
	return nil
}

// Get devuelve un tenant del caché. Si no está, lo carga de DB y lo cachea.
func (m *Manager) Get(ctx context.Context, tenantID uuid.UUID) (*Record, error) {
	m.mu.RLock()
	r, ok := m.items[tenantID]
	m.mu.RUnlock()
	if ok {
		return r, nil
	}
	return m.Refresh(ctx, tenantID)
}

// Refresh fuerza una lectura desde DB y reemplaza el caché.
func (m *Manager) Refresh(ctx context.Context, tenantID uuid.UUID) (*Record, error) {
	r, err := m.store.ByID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.items[tenantID] = r
	m.mu.Unlock()
	return r, nil
}

// All devuelve un snapshot de todos los tenants cacheados.
func (m *Manager) All() []*Record {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Record, 0, len(m.items))
	for _, r := range m.items {
		out = append(out, r)
	}
	return out
}

func (m *Manager) Store() *Store { return m.store }

var ErrNoMember = errors.New("tenant: el usuario no es miembro de este tenant")
