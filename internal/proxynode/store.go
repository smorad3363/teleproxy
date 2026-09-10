package proxynode

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/mattn/go-sqlite3"
)

var (
	ErrNotFound = errors.New("proxy node not found")
	ErrConflict = errors.New("proxy node already exists")
)

type Type string

const (
	TypeProxy Type = "proxy"
	TypeRelay Type = "relay"
)

type Node struct {
	ID                  int64     `json:"id"`
	Type                Type      `json:"node_type"`
	Name                string    `json:"name"`
	Region              string    `json:"region"`
	Host                string    `json:"host"`
	PublicHost          string    `json:"public_host"`
	MTProtoPort         int       `json:"mtproto_port"`
	InternalAPIEndpoint string    `json:"internal_api_endpoint"`
	Enabled             bool      `json:"enabled"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

type CreateNode struct {
	Type                Type
	Name                string
	Region              string
	Host                string
	PublicHost          string
	MTProtoPort         int
	InternalAPIEndpoint string
	Enabled             bool
}

func Create(ctx context.Context, db *sql.DB, input CreateNode, now time.Time) (Node, error) {
	if db == nil {
		return Node{}, fmt.Errorf("proxy node database is required")
	}
	if now.IsZero() {
		return Node{}, fmt.Errorf("proxy node time is required")
	}
	normalized, err := normalizeInput(input)
	if err != nil {
		return Node{}, err
	}
	now = now.UTC().Truncate(time.Second)
	result, err := db.ExecContext(ctx, `
INSERT INTO proxy_nodes(
    node_type, name, region, host, public_host, mtproto_port, internal_api_endpoint, enabled, created_at, updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(normalized.Type), normalized.Name, normalized.Region, normalized.Host, normalized.PublicHost,
		normalized.MTProtoPort, normalized.InternalAPIEndpoint, boolInt(normalized.Enabled), now.Unix(), now.Unix(),
	)
	if isUniqueConstraint(err) {
		return Node{}, ErrConflict
	}
	if err != nil {
		return Node{}, fmt.Errorf("create proxy node: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return Node{}, fmt.Errorf("read proxy node id: %w", err)
	}
	return Get(ctx, db, id)
}

func Get(ctx context.Context, db *sql.DB, id int64) (Node, error) {
	if db == nil {
		return Node{}, fmt.Errorf("proxy node database is required")
	}
	if id <= 0 {
		return Node{}, ErrNotFound
	}
	node, err := scanNode(db.QueryRowContext(ctx, `
SELECT id, node_type, name, region, host, public_host, mtproto_port, internal_api_endpoint, enabled, created_at, updated_at
FROM proxy_nodes WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Node{}, ErrNotFound
	}
	return node, err
}

func List(ctx context.Context, db *sql.DB) ([]Node, error) {
	if db == nil {
		return nil, fmt.Errorf("proxy node database is required")
	}
	rows, err := db.QueryContext(ctx, `
SELECT id, node_type, name, region, host, public_host, mtproto_port, internal_api_endpoint, enabled, created_at, updated_at
FROM proxy_nodes
ORDER BY id ASC`)
	if err != nil {
		return nil, fmt.Errorf("list proxy nodes: %w", err)
	}
	defer rows.Close()

	nodes := make([]Node, 0)
	for rows.Next() {
		node, err := scanNode(rows)
		if err != nil {
			return nil, err
		}
		nodes = append(nodes, node)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate proxy nodes: %w", err)
	}
	return nodes, nil
}

func ValidateInput(input CreateNode) error {
	_, err := normalizeInput(input)
	return err
}

func normalizeInput(input CreateNode) (CreateNode, error) {
	nodeType := Type(strings.ToLower(strings.TrimSpace(string(input.Type))))
	if nodeType != TypeProxy && nodeType != TypeRelay {
		return CreateNode{}, fmt.Errorf("proxy node type is invalid")
	}
	name := strings.TrimSpace(input.Name)
	if !utf8.ValidString(name) || len(name) < 1 || len(name) > 128 {
		return CreateNode{}, fmt.Errorf("proxy node name must contain between 1 and 128 UTF-8 bytes")
	}
	region := strings.TrimSpace(input.Region)
	if !utf8.ValidString(region) || len(region) < 1 || len(region) > 128 {
		return CreateNode{}, fmt.Errorf("proxy node region must contain between 1 and 128 UTF-8 bytes")
	}
	host, err := normalizeHost(input.Host)
	if err != nil {
		return CreateNode{}, fmt.Errorf("proxy node host: %w", err)
	}
	publicHost, err := normalizeHost(input.PublicHost)
	if err != nil {
		return CreateNode{}, fmt.Errorf("proxy node public host: %w", err)
	}
	if input.MTProtoPort < 1 || input.MTProtoPort > 65535 {
		return CreateNode{}, fmt.Errorf("proxy node MTProto port is out of range")
	}
	apiEndpoint, err := normalizeAPIEndpoint(input.InternalAPIEndpoint)
	if err != nil {
		return CreateNode{}, err
	}
	input.Type = nodeType
	input.Name = name
	input.Region = region
	input.Host = host
	input.PublicHost = publicHost
	input.InternalAPIEndpoint = apiEndpoint
	return input, nil
}

func normalizeHost(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 255 || strings.ContainsAny(value, "/?#@ 	\r\n") {
		return "", fmt.Errorf("host is invalid")
	}
	if ip := net.ParseIP(value); ip != nil {
		return ip.String(), nil
	}
	value = strings.TrimSuffix(strings.ToLower(value), ".")
	if value == "" || len(value) > 253 {
		return "", fmt.Errorf("host is invalid")
	}
	labels := strings.Split(value, ".")
	for _, label := range labels {
		if len(label) < 1 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", fmt.Errorf("host is invalid")
		}
		for _, ch := range label {
			if (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '-' {
				continue
			}
			return "", fmt.Errorf("host is invalid")
		}
	}
	return value, nil
}

func normalizeAPIEndpoint(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 1024 {
		return "", fmt.Errorf("proxy node internal API endpoint is invalid")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.User != nil || parsed.Fragment != "" || parsed.RawQuery != "" || parsed.Hostname() == "" {
		return "", fmt.Errorf("proxy node internal API endpoint is invalid")
	}
	parsed.Scheme = strings.ToLower(parsed.Scheme)
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", fmt.Errorf("proxy node internal API endpoint must use http or https")
	}
	host, err := normalizeHost(parsed.Hostname())
	if err != nil {
		return "", fmt.Errorf("proxy node internal API endpoint host is invalid")
	}
	port := parsed.Port()
	if port != "" {
		portNumber, err := strconv.Atoi(port)
		if err != nil || portNumber < 1 || portNumber > 65535 {
			return "", fmt.Errorf("proxy node internal API endpoint port is invalid")
		}
		parsed.Host = net.JoinHostPort(host, port)
	} else if strings.Contains(host, ":") {
		parsed.Host = "[" + host + "]"
	} else {
		parsed.Host = host
	}
	return parsed.String(), nil
}

type scanner interface {
	Scan(...any) error
}

func scanNode(row scanner) (Node, error) {
	var node Node
	var nodeType string
	var enabled int
	var createdAt, updatedAt int64
	if err := row.Scan(
		&node.ID, &nodeType, &node.Name, &node.Region, &node.Host, &node.PublicHost,
		&node.MTProtoPort, &node.InternalAPIEndpoint, &enabled, &createdAt, &updatedAt,
	); err != nil {
		return Node{}, err
	}
	if enabled != 0 && enabled != 1 {
		return Node{}, fmt.Errorf("proxy node enabled state is invalid")
	}
	normalized, err := normalizeInput(CreateNode{
		Type: Type(nodeType), Name: node.Name, Region: node.Region, Host: node.Host, PublicHost: node.PublicHost,
		MTProtoPort: node.MTProtoPort, InternalAPIEndpoint: node.InternalAPIEndpoint, Enabled: enabled == 1,
	})
	if err != nil || string(normalized.Type) != nodeType || normalized.Name != node.Name || normalized.Region != node.Region ||
		normalized.Host != node.Host || normalized.PublicHost != node.PublicHost || normalized.InternalAPIEndpoint != node.InternalAPIEndpoint {
		return Node{}, fmt.Errorf("stored proxy node is invalid")
	}
	node.Type = normalized.Type
	node.Enabled = enabled == 1
	node.CreatedAt = time.Unix(createdAt, 0).UTC()
	node.UpdatedAt = time.Unix(updatedAt, 0).UTC()
	return node, nil
}

func isUniqueConstraint(err error) bool {
	if err == nil {
		return false
	}
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	return sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique || sqliteErr.ExtendedCode == sqlite3.ErrConstraintPrimaryKey
}

func boolInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
