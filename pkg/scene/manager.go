package scene

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
)

type Manager struct {
	currentScene Scene
}

func (m Manager) GetCurrentScene() Scene {
	return m.currentScene
}

func (m *Manager) Update(gs *GameState) error {
	if m.currentScene == nil {
		return nil
	}
	return m.currentScene.Update(gs)
}

func (m *Manager) Draw(screen *ebiten.Image, gs *GameState) {
	if m.currentScene == nil || gs == nil {
		return
	}
	m.currentScene.Draw(screen, gs)
}

func (m *Manager) ChangeScene(scene Scene, gs *GameState) error {
	if scene.Name() == m.currentScene.Name() {
		return nil
	}

	if err := m.currentScene.Unload(); err != nil {
		return fmt.Errorf("NextScene: m.currentScene.Unload: %w", err)
	}

	if err := scene.Load(gs); err != nil {
		return fmt.Errorf("NextScene: Scene.Load: %w", err)
	}

	m.currentScene = scene
	return nil
}

func NewManager(scene Scene) Manager {
	return Manager{
		currentScene: scene,
	}
}
