package memory

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cloudwego/eino/schema"
)

// 会话元数据
type SessionMeta struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
}

type Session struct {
	ID 	  string
	CreatedAt time.Time

	filePath string					 // 会话记忆路径
	mu       sync.Mutex
	messages []*schema.Message       // 消息列表
	interruptID string        // 会话中断 ID
	msgIdx int					// 中断时的消息索引
}

// SetInterruptID 存储中断 ID，以便 approve 端点可以恢复它
func (s *Session) SetInterruptID(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.interruptID = id
}

// GetInterruptID 返回存储的中断 ID，如果没有则返回空字符串
func (s *Session) GetInterruptID() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.interruptID
}

// SetMsgIdx 存储中断时的消息索引，以便恢复时可以从该位置继续
func (s *Session) SetMsgIdx(idx int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.msgIdx = idx
}

// GetMsgIdx 返回存储的消息索引
func (s *Session) GetMsgIdx() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.msgIdx
}

// Append 添加消息到内存并持久化到磁盘
func (s *Session) Append(msg *schema.Message) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 合并消息到内存
	s.messages = append(s.messages, msg)

	// 将消息持久化到磁盘
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return nil
}

// GetMessages 返回磁盘中存储的消息
func (s *Session) GetMessages() []*schema.Message {
	s.mu.Lock()
	defer s.mu.Unlock()

	msg := make([]*schema.Message, len(s.messages))
	copy(msg, s.messages)
	return msg
}

// GetTitle 从用户消息派生标题
func (s *Session) GetTitle() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, msg := range s.messages {
		if msg.Role == schema.User && msg.Content != "" {
			title := msg.Content
			if len([]rune(title)) > 60 {
				title = string([]rune(title)[:60]) + "..."
			}
			return title
		}
	}

	return "新会话"
}

// 会话管理器
type Store struct {
	sessions map[string]*Session // 会话缓存
	mu       sync.Mutex
	dir 	 string
}

// NewStore 创建新的会话存储
func NewStore(dir string) (*Store, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Store{
		sessions: make(map[string]*Session),
		dir:      dir,
	}, nil
}

// sessionHeader 每个会话的第一行 JSONL 数据
type SessionHeader struct {
	Type      string    `json:"type"`
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"created_at"`
}

// GetSession 获取或创建会话
func (s *Store) GetSession(id string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 命中缓存
	if session, exists := s.sessions[id]; exists {
		return session, nil
	}

	filepath := filepath.Join(s.dir, id+".jsonl")

	var session *Session
	if _, exists := os.Stat(filepath);os.IsNotExist(exists){ // 如果文件不存在，创建新会话
		header := SessionHeader{
			Type:      "session",
			ID:        id,
			CreatedAt: time.Now(),
		}
		data, err := json.Marshal(header)
		if err != nil {
			return nil, err
		}

		// 写入文件
		if err := os.WriteFile(filepath, append(data, '\n'), 0o644); err != nil {
			return nil, err
		}
		session = &Session{
			ID:        id,
			CreatedAt: header.CreatedAt,
			filePath:  filepath,
			messages:  make([]*schema.Message, 0),
		}
	}else { // 从磁盘加载会话
		file, err := os.Open(filepath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)

		// 读取sessionHeader
		if !scanner.Scan() {
			return nil, fmt.Errorf("会话内容为空: %s", filepath)
		}
		var header SessionHeader
		if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
			return nil, fmt.Errorf("解析会话头部失败: %w", err)
		}

		// 保存SessionHeader
		session = &Session{
			ID:        header.ID,
			CreatedAt: header.CreatedAt,
			filePath:  filepath,
			messages:  make([]*schema.Message, 0),
		}

		// 读取消息
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var msg schema.Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}
			session.messages = append(session.messages, &msg)
		}
	}

	// 将会话加入缓存
	s.sessions[id] = session

	return session, nil
}

// ListSessions 列出所有会话元数据
func (s *Store) ListSessions() ([]SessionMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	files, err := os.ReadDir(s.dir)
	if err != nil {
		return nil, err
	}

	var sessionMetas []SessionMeta
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".jsonl") { // 只处理 .jsonl 文件
			continue
		}
		id := strings.TrimSuffix(file.Name(), ".jsonl")

		// 命中缓存
		if session, exists := s.sessions[id]; exists {
			sessionMetas = append(sessionMetas, SessionMeta{
				ID:        session.ID,
				Title:     session.GetTitle(),
				CreatedAt: session.CreatedAt,
			})
			continue
		}

		// 从磁盘加载会话（如果加载失败，则跳过该会话)
		filepath := filepath.Join(s.dir, file.Name())
		localFile, err := os.Open(filepath)
		if err != nil {
			continue
		}
		defer localFile.Close()

		scanner := bufio.NewScanner(localFile)

		// 读取sessionHeader
		if !scanner.Scan() {
			continue
		}
		var header SessionHeader
		if err := json.Unmarshal(scanner.Bytes(), &header); err != nil {
			continue
		}

		// 保存SessionHeader
		session := &Session{
			ID:        header.ID,
			CreatedAt: header.CreatedAt,
			filePath:  filepath,
			messages:  make([]*schema.Message, 0),
		}

		// 读取消息
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line == "" {
				continue
			}
			var msg schema.Message
			if err := json.Unmarshal([]byte(line), &msg); err != nil {
				continue
			}
			session.messages = append(session.messages, &msg)
		}

		// 添加加载的元数据
		sessionMetas = append(sessionMetas, SessionMeta{
			ID:        session.ID,
			Title:     session.GetTitle(),
			CreatedAt: session.CreatedAt,
		})
	}

	return sessionMetas, nil
}

// DeleteSession 删除会话
func (s *Store) DeleteSession(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 从缓存中删除
	delete(s.sessions, id)

	// 从磁盘删除
	filepath := filepath.Join(s.dir, id+".jsonl")
	if err := os.Remove(filepath); err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
