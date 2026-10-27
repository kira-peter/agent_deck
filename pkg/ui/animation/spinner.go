package animation

import (
	"math/rand"
	"sync"
	"time"
)

// AnimationType 动画类型
type AnimationType int

const (
	AnimSnowflake AnimationType = iota // 雪花闪烁
	AnimBraille                        // Braille 点阵旋转
	AnimDots                           // 跳动的点（移动）
	AnimBouncing                       // 弹跳球
	AnimSnake                          // 蛇形移动
	AnimArrow                          // 箭头旋转
	AnimWave                           // 波浪移动
	AnimMoon                           // 月相旋转
	AnimSquare                         // 方块翻转
)

// Spinner represents an animated spinner
type Spinner struct {
	frames       []rune
	currentFrame int
	ticker       *time.Ticker
	mu           sync.RWMutex
	running      bool
	animType     AnimationType
}

// 九种动画的帧
var animationFrames = map[AnimationType][]rune{
	AnimSnowflake: {'*', '✻', '❅', '✼', '❆', '❉', '❆', '✼', '❅', '✻'},
	AnimBraille:   {'⠋', '⠙', '⠹', '⠸', '⠼', '⠴', '⠦', '⠧', '⠇', '⠏'},
	AnimDots:      {'⣾', '⣽', '⣻', '⢿', '⡿', '⣟', '⣯', '⣷'},
	AnimBouncing:  {'˙', '·', '∘', '○', '◯', '○', '∘', '·'},
	AnimSnake:     {'⠁', '⠂', '⠄', '⡀', '⢀', '⠠', '⠐', '⠈'},
	AnimArrow:     {'←', '↖', '↑', '↗', '→', '↘', '↓', '↙'},
	AnimWave:      {'◜', '◝', '◞', '◟'},
	AnimMoon:      {'◐', '◓', '◑', '◒'},
	AnimSquare:    {'◰', '◳', '◲', '◱'},
}

// NewSpinner creates a new spinner with default animation
func NewSpinner() *Spinner {
	return NewSpinnerWithType(AnimSnowflake)
}

// NewSpinnerWithType creates a new spinner with specified animation type
func NewSpinnerWithType(animType AnimationType) *Spinner {
	frames := animationFrames[animType]
	return &Spinner{
		frames:       frames,
		currentFrame: 0,
		animType:     animType,
	}
}

// RandomAnimationType returns a random animation type
func RandomAnimationType() AnimationType {
	types := []AnimationType{
		AnimSnowflake, AnimBraille, AnimDots, AnimBouncing, AnimSnake,
		AnimArrow, AnimWave, AnimMoon, AnimSquare,
	}
	return types[rand.Intn(len(types))]
}

// RandomAnimationTypeExcept returns a random animation type excluding the specified one
func RandomAnimationTypeExcept(exclude AnimationType) AnimationType {
	types := []AnimationType{
		AnimSnowflake, AnimBraille, AnimDots, AnimBouncing, AnimSnake,
		AnimArrow, AnimWave, AnimMoon, AnimSquare,
	}

	// Filter out the excluded type
	var available []AnimationType
	for _, t := range types {
		if t != exclude {
			available = append(available, t)
		}
	}

	// If no available types (shouldn't happen), return a random one
	if len(available) == 0 {
		return types[rand.Intn(len(types))]
	}

	return available[rand.Intn(len(available))]
}

// GetAnimationType returns the current animation type
func (s *Spinner) GetAnimationType() AnimationType {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.animType
}

// SetAnimationType changes the animation type
func (s *Spinner) SetAnimationType(animType AnimationType) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.animType = animType
	s.frames = animationFrames[animType]
	s.currentFrame = 0
}

// Start starts the spinner animation
func (s *Spinner) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return
	}

	s.running = true
	s.ticker = time.NewTicker(80 * time.Millisecond)

	go func() {
		for range s.ticker.C {
			s.mu.Lock()
			if !s.running {
				s.mu.Unlock()
				return
			}
			s.currentFrame = (s.currentFrame + 1) % len(s.frames)
			s.mu.Unlock()
		}
	}()
}

// Stop stops the spinner animation
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	s.running = false
	if s.ticker != nil {
		s.ticker.Stop()
		s.ticker = nil
	}
}

// GetFrame returns the current animation frame
func (s *Spinner) GetFrame() rune {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.frames[s.currentFrame]
}

// IsRunning returns whether the spinner is running
func (s *Spinner) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}
