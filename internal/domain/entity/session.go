package entity

import "time"

// Session 会话实体
type Session struct {
	ID        string    `json:"id"`
	Messages  []Message `json:"messages"`
	Model     string    `json:"model"`
	Provider  string    `json:"provider"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewSession 创建新会话
func NewSession(model, provider string) *Session {
	now := time.Now()
	return &Session{
		ID:        generateID(),
		Messages:  []Message{},
		Model:     model,
		Provider:  provider,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// AddMessage 添加消息
func (s *Session) AddMessage(msg Message) {
	s.Messages = append(s.Messages, msg)
	s.UpdatedAt = time.Now()
}

// Clear 清空消息
func (s *Session) Clear() {
	s.Messages = []Message{}
	s.UpdatedAt = time.Now()
}

// LastMessage 获取最后一条消息
func (s *Session) LastMessage() *Message {
	if len(s.Messages) == 0 {
		return nil
	}
	return &s.Messages[len(s.Messages)-1]
}

// SetModel 设置模型
func (s *Session) SetModel(model string) {
	s.Model = model
	s.UpdatedAt = time.Now()
}
