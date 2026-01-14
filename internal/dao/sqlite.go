// Package dao provides data access operations for SQLite database.
package dao

import (
	"database/sql"
	"encoding/json"
	"errors"
	"sync"
	"time"

	// 使用纯 Go 实现的 SQLite 驱动，无需 CGO 支持
	// 解决 Docker 容器中 CGO_ENABLED=0 导致的 go-sqlite3 无法工作问题
	"go.uber.org/zap"
	_ "modernc.org/sqlite"
)

// DB is the global database instance.
var (
	db     *sql.DB
	dbOnce sync.Once
	logger *zap.Logger
)

// InitDB initializes the SQLite database.
func InitDB(dbPath string, log *zap.Logger) error {
	var initErr error
	dbOnce.Do(func() {
		logger = log
		var err error
		// 使用 modernc.org/sqlite 驱动，驱动名为 "sqlite"
		// 支持 WAL 模式和忙等待超时
		db, err = sql.Open("sqlite", dbPath+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
		if err != nil {
			initErr = err
			return
		}

		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)

		if err := createTables(); err != nil {
			initErr = err
			return
		}

		if err := seedDefaultData(); err != nil {
			initErr = err
			return
		}
	})
	return initErr
}

// GetDB returns the database instance.
func GetDB() *sql.DB {
	return db
}

// CloseDB closes the database connection.
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

func createTables() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			email TEXT,
			role TEXT DEFAULT 'user',
			is_active INTEGER DEFAULT 1,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			last_login_at DATETIME
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id INTEGER NOT NULL,
			ip TEXT,
			user_agent TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			expires_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS personal_access_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			token_hash TEXT UNIQUE NOT NULL,
			scopes TEXT,
			expires_at DATETIME,
			last_used_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (user_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS access_attempts (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			ip_address TEXT,
			user_agent TEXT,
			user_id INTEGER,
			action TEXT,
			resource TEXT,
			status TEXT,
			error_msg TEXT,
			blockchain_hash TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS system_status (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			is_locked INTEGER DEFAULT 0,
			lock_reason TEXT,
			lock_type TEXT,
			locked_at DATETIME,
			locked_by_ip TEXT,
			locked_by_user TEXT,
			unlock_at DATETIME,
			require_manual INTEGER DEFAULT 1
		)`,
		`CREATE TABLE IF NOT EXISTS organizations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL,
			display_name TEXT,
			owner_id INTEGER NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (owner_id) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS org_members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			org_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			role TEXT DEFAULT 'member',
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (org_id) REFERENCES organizations(id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			UNIQUE(org_id, user_id)
		)`,
		`CREATE TABLE IF NOT EXISTS share_links (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			code TEXT UNIQUE NOT NULL,
			image_ref TEXT NOT NULL,
			created_by INTEGER NOT NULL,
			password_hash TEXT,
			max_usage INTEGER DEFAULT 0,
			usage_count INTEGER DEFAULT 0,
			expires_at DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (created_by) REFERENCES users(id)
		)`,
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,
			level TEXT,
			event TEXT,
			user_id INTEGER,
			username TEXT,
			ip_address TEXT,
			resource TEXT,
			action TEXT,
			status TEXT,
			details TEXT,
			blockchain_hash TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at)`,
		`CREATE INDEX IF NOT EXISTS idx_access_attempts_ip ON access_attempts(ip_address)`,
		`CREATE INDEX IF NOT EXISTS idx_access_attempts_created ON access_attempts(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_timestamp ON audit_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_logs_event ON audit_logs(event)`,
		`CREATE INDEX IF NOT EXISTS idx_share_links_code ON share_links(code)`,
	}

	for _, schema := range schemas {
		if _, err := db.Exec(schema); err != nil {
			return err
		}
	}

	return nil
}

func seedDefaultData() error {
	// Insert default system status
	_, err := db.Exec(`INSERT OR IGNORE INTO system_status (id, is_locked) VALUES (1, 0)`)
	if err != nil {
		return err
	}

	// Check if admin user exists
	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM users WHERE username = 'admin'`).Scan(&count)
	if err != nil {
		return err
	}

	if count == 0 {
		// Default password: admin123 (should be changed on first login)
		// bcrypt hash of "admin123"
		hash := "$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy"
		_, err = db.Exec(`INSERT INTO users (username, password_hash, email, role, is_active) VALUES (?, ?, ?, ?, ?)`,
			"admin", hash, "admin@localhost", "admin", 1)
		if err != nil {
			return err
		}
	}

	return nil
}

// User operations

// GetUserByUsername retrieves a user by username.
func GetUserByUsername(username string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, username, password_hash, email, role, is_active, created_at, updated_at, last_login_at
		FROM users WHERE username = ?
	`, username).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByEmail retrieves a user by email.
func GetUserByEmail(email string) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, username, password_hash, email, role, is_active, created_at, updated_at, last_login_at
		FROM users WHERE email = ?
	`, email).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// GetUserByID retrieves a user by ID.
func GetUserByID(id int64) (*User, error) {
	user := &User{}
	err := db.QueryRow(`
		SELECT id, username, password_hash, email, role, is_active, created_at, updated_at, last_login_at
		FROM users WHERE id = ?
	`, id).Scan(
		&user.ID, &user.Username, &user.PasswordHash, &user.Email,
		&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}

// CreateUser creates a new user.
func CreateUser(user *User) error {
	result, err := db.Exec(`
		INSERT INTO users (username, password_hash, email, role, is_active)
		VALUES (?, ?, ?, ?, ?)
	`, user.Username, user.PasswordHash, user.Email, user.Role, user.IsActive)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	user.ID = id
	return nil
}

// UpdateUser updates a user.
func UpdateUser(user *User) error {
	_, err := db.Exec(`
		UPDATE users SET email = ?, role = ?, is_active = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, user.Email, user.Role, user.IsActive, user.ID)
	return err
}

// UpdateUserPassword updates a user's password.
func UpdateUserPassword(userID int64, passwordHash string) error {
	_, err := db.Exec(`UPDATE users SET password_hash = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		passwordHash, userID)
	return err
}

// UpdateUserLastLogin updates the last login time.
func UpdateUserLastLogin(userID int64) error {
	_, err := db.Exec(`UPDATE users SET last_login_at = CURRENT_TIMESTAMP WHERE id = ?`, userID)
	return err
}

// ListUsers lists all users.
func ListUsers(page, pageSize int) ([]*User, int, error) {
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := db.Query(`
		SELECT id, username, password_hash, email, role, is_active, created_at, updated_at, last_login_at
		FROM users ORDER BY id LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		user := &User{}
		err := rows.Scan(
			&user.ID, &user.Username, &user.PasswordHash, &user.Email,
			&user.Role, &user.IsActive, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt,
		)
		if err != nil {
			return nil, 0, err
		}
		users = append(users, user)
	}
	return users, total, nil
}

// DeleteUser deletes a user.
func DeleteUser(id int64) error {
	_, err := db.Exec(`DELETE FROM users WHERE id = ?`, id)
	return err
}

// Session operations

// CreateSession creates a new session.
func CreateSession(session *Session) error {
	_, err := db.Exec(`
		INSERT INTO sessions (id, user_id, ip, user_agent, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, session.ID, session.UserID, session.IP, session.UserAgent, session.ExpiresAt)
	return err
}

// GetSession retrieves a session by ID.
func GetSession(id string) (*Session, error) {
	session := &Session{}
	err := db.QueryRow(`
		SELECT id, user_id, ip, user_agent, created_at, expires_at
		FROM sessions WHERE id = ?
	`, id).Scan(&session.ID, &session.UserID, &session.IP, &session.UserAgent, &session.CreatedAt, &session.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

// GetSessionByUserID retrieves a session by user ID.
func GetSessionByUserID(userID int64) (*Session, error) {
	session := &Session{}
	err := db.QueryRow(`
		SELECT id, user_id, ip, user_agent, created_at, expires_at
		FROM sessions WHERE user_id = ? AND expires_at > CURRENT_TIMESTAMP
		ORDER BY created_at DESC LIMIT 1
	`, userID).Scan(&session.ID, &session.UserID, &session.IP, &session.UserAgent, &session.CreatedAt, &session.ExpiresAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return session, nil
}

// DeleteSession deletes a session.
func DeleteSession(id string) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE id = ?`, id)
	return err
}

// DeleteUserSessions deletes all sessions for a user.
func DeleteUserSessions(userID int64) error {
	_, err := db.Exec(`DELETE FROM sessions WHERE user_id = ?`, userID)
	return err
}

// CleanExpiredSessions removes expired sessions.
func CleanExpiredSessions() error {
	_, err := db.Exec(`DELETE FROM sessions WHERE expires_at < CURRENT_TIMESTAMP`)
	return err
}

// Token operations

// CreateToken creates a new personal access token.
func CreateToken(token *PersonalAccessToken) error {
	scopesJSON, _ := json.Marshal(token.Scopes)
	result, err := db.Exec(`
		INSERT INTO personal_access_tokens (user_id, name, token_hash, scopes, expires_at)
		VALUES (?, ?, ?, ?, ?)
	`, token.UserID, token.Name, token.TokenHash, string(scopesJSON), token.ExpiresAt)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	token.ID = id
	return nil
}

// GetTokenByHash retrieves a token by its hash.
func GetTokenByHash(hash string) (*PersonalAccessToken, error) {
	token := &PersonalAccessToken{}
	var scopesJSON string
	err := db.QueryRow(`
		SELECT id, user_id, name, token_hash, scopes, expires_at, last_used_at, created_at
		FROM personal_access_tokens WHERE token_hash = ?
	`, hash).Scan(
		&token.ID, &token.UserID, &token.Name, &token.TokenHash,
		&scopesJSON, &token.ExpiresAt, &token.LastUsedAt, &token.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(scopesJSON), &token.Scopes)
	return token, nil
}

// ListUserTokens lists all tokens for a user.
func ListUserTokens(userID int64) ([]*PersonalAccessToken, error) {
	rows, err := db.Query(`
		SELECT id, user_id, name, scopes, expires_at, last_used_at, created_at
		FROM personal_access_tokens WHERE user_id = ? ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []*PersonalAccessToken
	for rows.Next() {
		token := &PersonalAccessToken{}
		var scopesJSON string
		err := rows.Scan(&token.ID, &token.UserID, &token.Name, &scopesJSON, &token.ExpiresAt, &token.LastUsedAt, &token.CreatedAt)
		if err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(scopesJSON), &token.Scopes)
		tokens = append(tokens, token)
	}
	return tokens, nil
}

// UpdateTokenLastUsed updates the last used time of a token.
func UpdateTokenLastUsed(id int64) error {
	_, err := db.Exec(`UPDATE personal_access_tokens SET last_used_at = CURRENT_TIMESTAMP WHERE id = ?`, id)
	return err
}

// DeleteToken deletes a token.
func DeleteToken(id int64) error {
	_, err := db.Exec(`DELETE FROM personal_access_tokens WHERE id = ?`, id)
	return err
}

// Access attempt operations

// CreateAccessAttempt creates a new access attempt record.
func CreateAccessAttempt(attempt *AccessAttempt) error {
	result, err := db.Exec(`
		INSERT INTO access_attempts (ip_address, user_agent, user_id, action, resource, status, error_msg, blockchain_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, attempt.IPAddress, attempt.UserAgent, attempt.UserID, attempt.Action, attempt.Resource, attempt.Status, attempt.ErrorMsg, attempt.BlockchainHash)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	attempt.ID = id
	return nil
}

// UpdateAccessAttemptHash updates the blockchain hash of an access attempt.
func UpdateAccessAttemptHash(id int64, hash string) error {
	_, err := db.Exec(`UPDATE access_attempts SET blockchain_hash = ? WHERE id = ?`, hash, id)
	return err
}

// GetAccessAttempts retrieves access attempts with pagination.
func GetAccessAttempts(page, pageSize int, ip string) ([]*AccessAttempt, int, error) {
	var total int
	var args []interface{}
	query := `SELECT COUNT(*) FROM access_attempts`
	if ip != "" {
		query += ` WHERE ip_address = ?`
		args = append(args, ip)
	}
	err := db.QueryRow(query, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query = `SELECT id, ip_address, user_agent, user_id, action, resource, status, error_msg, blockchain_hash, created_at
		FROM access_attempts`
	if ip != "" {
		query += ` WHERE ip_address = ?`
	}
	query += ` ORDER BY created_at DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var attempts []*AccessAttempt
	for rows.Next() {
		a := &AccessAttempt{}
		err := rows.Scan(&a.ID, &a.IPAddress, &a.UserAgent, &a.UserID, &a.Action, &a.Resource, &a.Status, &a.ErrorMsg, &a.BlockchainHash, &a.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		attempts = append(attempts, a)
	}
	return attempts, total, nil
}

// System status operations

// GetSystemStatus retrieves the system lock status.
func GetSystemStatus() (*LockStatus, error) {
	status := &LockStatus{}
	err := db.QueryRow(`
		SELECT is_locked, lock_reason, lock_type, locked_at, locked_by_ip, locked_by_user, unlock_at, require_manual
		FROM system_status WHERE id = 1
	`).Scan(&status.IsLocked, &status.LockReason, &status.LockType, &status.LockedAt, &status.LockedByIP, &status.LockedByUser, &status.UnlockAt, &status.RequireManual)
	if err == sql.ErrNoRows {
		return &LockStatus{}, nil
	}
	if err != nil {
		return nil, err
	}
	return status, nil
}

// UpdateSystemStatus updates the system lock status.
func UpdateSystemStatus(status *LockStatus) error {
	_, err := db.Exec(`
		UPDATE system_status SET is_locked = ?, lock_reason = ?, lock_type = ?, locked_at = ?, 
		locked_by_ip = ?, locked_by_user = ?, unlock_at = ?, require_manual = ? WHERE id = 1
	`, status.IsLocked, status.LockReason, status.LockType, status.LockedAt, status.LockedByIP, status.LockedByUser, status.UnlockAt, status.RequireManual)
	return err
}

// Organization operations

// CreateOrganization creates a new organization.
func CreateOrganization(org *Organization) error {
	result, err := db.Exec(`
		INSERT INTO organizations (name, display_name, owner_id)
		VALUES (?, ?, ?)
	`, org.Name, org.DisplayName, org.OwnerID)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	org.ID = id
	return nil
}

// GetOrganization retrieves an organization by ID.
func GetOrganization(id int64) (*Organization, error) {
	org := &Organization{}
	err := db.QueryRow(`
		SELECT id, name, display_name, owner_id, created_at, updated_at
		FROM organizations WHERE id = ?
	`, id).Scan(&org.ID, &org.Name, &org.DisplayName, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return org, nil
}

// GetOrganizationByName retrieves an organization by name.
func GetOrganizationByName(name string) (*Organization, error) {
	org := &Organization{}
	err := db.QueryRow(`
		SELECT id, name, display_name, owner_id, created_at, updated_at
		FROM organizations WHERE name = ?
	`, name).Scan(&org.ID, &org.Name, &org.DisplayName, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return org, nil
}

// ListOrganizations lists all organizations.
func ListOrganizations(page, pageSize int) ([]*Organization, int, error) {
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM organizations`).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := db.Query(`
		SELECT id, name, display_name, owner_id, created_at, updated_at
		FROM organizations ORDER BY name LIMIT ? OFFSET ?
	`, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org := &Organization{}
		err := rows.Scan(&org.ID, &org.Name, &org.DisplayName, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)
		if err != nil {
			return nil, 0, err
		}
		orgs = append(orgs, org)
	}
	return orgs, total, nil
}

// ListUserOrganizations lists organizations for a user.
func ListUserOrganizations(userID int64) ([]*Organization, error) {
	rows, err := db.Query(`
		SELECT o.id, o.name, o.display_name, o.owner_id, o.created_at, o.updated_at
		FROM organizations o
		LEFT JOIN org_members m ON o.id = m.org_id
		WHERE o.owner_id = ? OR m.user_id = ?
		GROUP BY o.id
	`, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []*Organization
	for rows.Next() {
		org := &Organization{}
		err := rows.Scan(&org.ID, &org.Name, &org.DisplayName, &org.OwnerID, &org.CreatedAt, &org.UpdatedAt)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, org)
	}
	return orgs, nil
}

// UpdateOrganization updates an organization.
func UpdateOrganization(org *Organization) error {
	_, err := db.Exec(`
		UPDATE organizations SET display_name = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?
	`, org.DisplayName, org.ID)
	return err
}

// DeleteOrganization deletes an organization.
func DeleteOrganization(id int64) error {
	_, err := db.Exec(`DELETE FROM org_members WHERE org_id = ?`, id)
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM organizations WHERE id = ?`, id)
	return err
}

// AddOrgMember adds a member to an organization.
func AddOrgMember(orgID, userID int64, role string) error {
	_, err := db.Exec(`INSERT OR REPLACE INTO org_members (org_id, user_id, role) VALUES (?, ?, ?)`, orgID, userID, role)
	return err
}

// RemoveOrgMember removes a member from an organization.
func RemoveOrgMember(orgID, userID int64) error {
	_, err := db.Exec(`DELETE FROM org_members WHERE org_id = ? AND user_id = ?`, orgID, userID)
	return err
}

// GetOrgMembers retrieves members of an organization.
func GetOrgMembers(orgID int64) ([]*OrgMember, error) {
	rows, err := db.Query(`
		SELECT m.id, m.org_id, m.user_id, m.role, m.created_at, u.username
		FROM org_members m
		JOIN users u ON m.user_id = u.id
		WHERE m.org_id = ?
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*OrgMember
	for rows.Next() {
		m := &OrgMember{}
		err := rows.Scan(&m.ID, &m.OrgID, &m.UserID, &m.Role, &m.CreatedAt, &m.Username)
		if err != nil {
			return nil, err
		}
		members = append(members, m)
	}
	return members, nil
}

// Share link operations

// CreateShareLink creates a new share link.
func CreateShareLink(link *ShareLink) error {
	result, err := db.Exec(`
		INSERT INTO share_links (code, image_ref, created_by, password_hash, max_usage, expires_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`, link.Code, link.ImageRef, link.CreatedBy, link.PasswordHash, link.MaxUsage, link.ExpiresAt)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	link.ID = id
	return nil
}

// GetShareLink retrieves a share link by code.
func GetShareLink(code string) (*ShareLink, error) {
	link := &ShareLink{}
	err := db.QueryRow(`
		SELECT id, code, image_ref, created_by, password_hash, max_usage, usage_count, expires_at, created_at
		FROM share_links WHERE code = ?
	`, code).Scan(&link.ID, &link.Code, &link.ImageRef, &link.CreatedBy, &link.PasswordHash, &link.MaxUsage, &link.UsageCount, &link.ExpiresAt, &link.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return link, nil
}

// ListShareLinks lists share links created by a user.
func ListShareLinks(userID int64, page, pageSize int) ([]*ShareLink, int, error) {
	var total int
	err := db.QueryRow(`SELECT COUNT(*) FROM share_links WHERE created_by = ?`, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	rows, err := db.Query(`
		SELECT id, code, image_ref, created_by, max_usage, usage_count, expires_at, created_at
		FROM share_links WHERE created_by = ? ORDER BY created_at DESC LIMIT ? OFFSET ?
	`, userID, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var links []*ShareLink
	for rows.Next() {
		link := &ShareLink{}
		err := rows.Scan(&link.ID, &link.Code, &link.ImageRef, &link.CreatedBy, &link.MaxUsage, &link.UsageCount, &link.ExpiresAt, &link.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		links = append(links, link)
	}
	return links, total, nil
}

// IncrementShareLinkUsage increments the usage count of a share link.
func IncrementShareLinkUsage(code string) error {
	_, err := db.Exec(`UPDATE share_links SET usage_count = usage_count + 1 WHERE code = ?`, code)
	return err
}

// DeleteShareLink deletes a share link.
func DeleteShareLink(id int64) error {
	_, err := db.Exec(`DELETE FROM share_links WHERE id = ?`, id)
	return err
}

// Audit log operations

// CreateAuditLog creates a new audit log entry.
func CreateAuditLog(log *AuditLog) error {
	detailsJSON, _ := json.Marshal(log.Details)
	result, err := db.Exec(`
		INSERT INTO audit_logs (level, event, user_id, username, ip_address, resource, action, status, details, blockchain_hash)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, log.Level, log.Event, log.UserID, log.Username, log.IPAddress, log.Resource, log.Action, log.Status, string(detailsJSON), log.BlockchainHash)
	if err != nil {
		return err
	}
	id, _ := result.LastInsertId()
	log.ID = id
	return nil
}

// GetAuditLogs retrieves audit logs with filters.
func GetAuditLogs(page, pageSize int, eventType string, startDate, endDate time.Time) ([]*AuditLog, int, error) {
	var total int
	var args []interface{}
	query := `SELECT COUNT(*) FROM audit_logs WHERE 1=1`

	if eventType != "" {
		query += ` AND event = ?`
		args = append(args, eventType)
	}
	if !startDate.IsZero() {
		query += ` AND timestamp >= ?`
		args = append(args, startDate)
	}
	if !endDate.IsZero() {
		query += ` AND timestamp <= ?`
		args = append(args, endDate)
	}

	err := db.QueryRow(query, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	query = `SELECT id, timestamp, level, event, user_id, username, ip_address, resource, action, status, details, blockchain_hash
		FROM audit_logs WHERE 1=1`

	if eventType != "" {
		query += ` AND event = ?`
	}
	if !startDate.IsZero() {
		query += ` AND timestamp >= ?`
	}
	if !endDate.IsZero() {
		query += ` AND timestamp <= ?`
	}
	query += ` ORDER BY timestamp DESC LIMIT ? OFFSET ?`
	args = append(args, pageSize, offset)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var logs []*AuditLog
	for rows.Next() {
		log := &AuditLog{}
		var detailsJSON sql.NullString
		err := rows.Scan(&log.ID, &log.Timestamp, &log.Level, &log.Event, &log.UserID, &log.Username, &log.IPAddress, &log.Resource, &log.Action, &log.Status, &detailsJSON, &log.BlockchainHash)
		if err != nil {
			return nil, 0, err
		}
		if detailsJSON.Valid {
			json.Unmarshal([]byte(detailsJSON.String), &log.Details)
		}
		logs = append(logs, log)
	}
	return logs, total, nil
}

// Model types for DAO

// User represents a user in the database.
type User struct {
	ID           int64
	Username     string
	PasswordHash string
	Email        sql.NullString
	Role         string
	IsActive     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LastLoginAt  sql.NullTime
}

// Session represents a session in the database.
type Session struct {
	ID        string
	UserID    int64
	IP        string
	UserAgent string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// PersonalAccessToken represents a token in the database.
type PersonalAccessToken struct {
	ID         int64
	UserID     int64
	Name       string
	TokenHash  string
	Scopes     []string
	ExpiresAt  sql.NullTime
	LastUsedAt sql.NullTime
	CreatedAt  time.Time
}

// AccessAttempt represents an access attempt in the database.
type AccessAttempt struct {
	ID             int64
	IPAddress      string
	UserAgent      string
	UserID         sql.NullInt64
	Action         string
	Resource       string
	Status         string
	ErrorMsg       string
	BlockchainHash string
	CreatedAt      time.Time
}

// LockStatus represents the system lock status.
type LockStatus struct {
	IsLocked      bool
	LockReason    sql.NullString
	LockType      sql.NullString
	LockedAt      sql.NullTime
	LockedByIP    sql.NullString
	LockedByUser  sql.NullString
	UnlockAt      sql.NullTime
	RequireManual bool
}

// Organization represents an organization in the database.
type Organization struct {
	ID          int64
	Name        string
	DisplayName string
	OwnerID     int64
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// OrgMember represents an organization member.
type OrgMember struct {
	ID        int64
	OrgID     int64
	UserID    int64
	Role      string
	Username  string
	CreatedAt time.Time
}

// ShareLink represents a share link in the database.
type ShareLink struct {
	ID           int64
	Code         string
	ImageRef     string
	CreatedBy    int64
	PasswordHash sql.NullString
	MaxUsage     int
	UsageCount   int
	ExpiresAt    sql.NullTime
	CreatedAt    time.Time
}

// AuditLog represents an audit log entry.
type AuditLog struct {
	ID             int64
	Timestamp      time.Time
	Level          string
	Event          string
	UserID         sql.NullInt64
	Username       sql.NullString
	IPAddress      string
	Resource       string
	Action         string
	Status         string
	Details        map[string]interface{}
	BlockchainHash string
}

// ErrNotFound is returned when a record is not found.
var ErrNotFound = errors.New("record not found")
