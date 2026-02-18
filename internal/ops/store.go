package ops

import (
	"github.com/jacksmith/tk/internal/model"
	"github.com/jacksmith/tk/internal/storage"
)

// Store defines the persistence interface required by business logic operations.
// The concrete implementation is storage.Storage, but this interface allows
// alternative backends (in-memory, HTTP, etc.) for testing and GUI use.
type Store interface {
	LoadProject(prefix string) (*model.ProjectFile, error)
	LoadProjectByID(id string) (*model.ProjectFile, error)
	SaveProject(p *model.ProjectFile) error
	ListProjects() ([]string, error)
	DeleteProject(prefix string) error
	ProjectExists(prefix string) bool
	LoadConfig() (*storage.Config, error)
}
