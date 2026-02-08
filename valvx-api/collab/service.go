package collab

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// Service implements BCF business logic.
type Service struct {
	DB *sql.DB
}

// NewService creates a new BCF service.
func NewService(db *sql.DB) *Service {
	return &Service{DB: db}
}

func (s *Service) ListTopics(ctx context.Context, projectID string, filters TopicFilters) ([]Topic, error) {
	query := `
		SELECT t.id, t.guid, t.title, t.description, t.priority, t.topic_type,
		       t.topic_status, t.stage, t.assigned_to, t.due_date, t.labels,
		       t.project_id, t.creator_id, t.modified_by, t.created_at, t.updated_at,
		       p.name as creator_name
		FROM collab_topic t
		LEFT JOIN iam_profile p ON p.id = t.creator_id
		WHERE t.project_id = $1`

	args := []interface{}{projectID}
	argIdx := 2

	if filters.Status != "" {
		query += fmt.Sprintf(" AND t.topic_status = $%d", argIdx)
		args = append(args, filters.Status)
		argIdx++
	}
	if filters.Priority != "" {
		query += fmt.Sprintf(" AND t.priority = $%d", argIdx)
		args = append(args, filters.Priority)
		argIdx++
	}
	if filters.AssignedTo != "" {
		query += fmt.Sprintf(" AND t.assigned_to = $%d", argIdx)
		args = append(args, filters.AssignedTo)
		argIdx++
	}

	query += " ORDER BY t.created_at DESC"

	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query topics: %w", err)
	}
	defer rows.Close()

	var topics []Topic
	for rows.Next() {
		var t Topic
		var labels []string
		err := rows.Scan(
			&t.ID, &t.GUID, &t.Title, &t.Description, &t.Priority, &t.TopicType,
			&t.TopicStatus, &t.Stage, &t.AssignedTo, &t.DueDate, pq.Array(&labels),
			&t.ProjectID, &t.CreatorID, &t.ModifiedBy, &t.CreatedAt, &t.UpdatedAt,
			&t.CreatorName,
		)
		if err != nil {
			return nil, fmt.Errorf("scan topic: %w", err)
		}
		t.Labels = labels

		// Fetch first viewpoint for snapshot preview
		vps, _ := s.listViewpoints(ctx, t.ID, 1)
		t.Viewpoints = vps

		topics = append(topics, t)
	}

	if topics == nil {
		topics = []Topic{}
	}
	return topics, nil
}

func (s *Service) GetTopic(ctx context.Context, topicID string) (*Topic, error) {
	var t Topic
	var labels []string

	err := s.DB.QueryRowContext(ctx, `
		SELECT t.id, t.guid, t.title, t.description, t.priority, t.topic_type,
		       t.topic_status, t.stage, t.assigned_to, t.due_date, t.labels,
		       t.project_id, t.creator_id, t.modified_by, t.created_at, t.updated_at,
		       p.name as creator_name
		FROM collab_topic t
		LEFT JOIN iam_profile p ON p.id = t.creator_id
		WHERE t.id = $1`, topicID,
	).Scan(
		&t.ID, &t.GUID, &t.Title, &t.Description, &t.Priority, &t.TopicType,
		&t.TopicStatus, &t.Stage, &t.AssignedTo, &t.DueDate, pq.Array(&labels),
		&t.ProjectID, &t.CreatorID, &t.ModifiedBy, &t.CreatedAt, &t.UpdatedAt,
		&t.CreatorName,
	)
	if err != nil {
		return nil, fmt.Errorf("get topic: %w", err)
	}
	t.Labels = labels

	t.Viewpoints, _ = s.listViewpoints(ctx, t.ID, 0)
	t.Comments, _ = s.ListComments(ctx, t.ID)

	return &t, nil
}

func (s *Service) CreateTopic(ctx context.Context, projectID, creatorID string, req CreateTopicRequest) (*Topic, error) {
	id := uuid.New().String()
	guid := uuid.New().String()
	now := time.Now().UTC()

	status := "Open"

	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO collab_topic (id, guid, title, description, priority, topic_type,
		    topic_status, assigned_to, due_date, labels, project_id, creator_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		id, guid, req.Title, req.Description, req.Priority, req.TopicType,
		status, req.AssignedTo, req.DueDate, pq.Array(req.Labels),
		projectID, creatorID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert topic: %w", err)
	}

	// Create viewpoint if provided
	if req.Viewpoint != nil {
		_, err := s.CreateViewpoint(ctx, id, *req.Viewpoint)
		if err != nil {
			return nil, fmt.Errorf("create viewpoint: %w", err)
		}
	}

	// Link file versions
	for _, fvID := range req.FileVersionIDs {
		_, err := s.DB.ExecContext(ctx, `
			INSERT INTO collab_topic_file (topic_id, file_version_id) VALUES ($1, $2)`,
			id, fvID,
		)
		if err != nil {
			return nil, fmt.Errorf("link file: %w", err)
		}
	}

	return s.GetTopic(ctx, id)
}

func (s *Service) UpdateTopic(ctx context.Context, topicID string, req CreateTopicRequest) (*Topic, error) {
	now := time.Now().UTC()

	_, err := s.DB.ExecContext(ctx, `
		UPDATE collab_topic SET
			title = COALESCE(NULLIF($2, ''), title),
			description = COALESCE($3, description),
			priority = COALESCE($4, priority),
			topic_type = COALESCE($5, topic_type),
			assigned_to = COALESCE($6, assigned_to),
			due_date = COALESCE($7, due_date),
			labels = COALESCE($8, labels),
			updated_at = $9
		WHERE id = $1`,
		topicID, req.Title, req.Description, req.Priority, req.TopicType,
		req.AssignedTo, req.DueDate, pq.Array(req.Labels), now,
	)
	if err != nil {
		return nil, fmt.Errorf("update topic: %w", err)
	}

	return s.GetTopic(ctx, topicID)
}

func (s *Service) DeleteTopic(ctx context.Context, topicID string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM collab_topic WHERE id = $1", topicID)
	return err
}

// --- Comments ---

func (s *Service) ListComments(ctx context.Context, topicID string) ([]Comment, error) {
	rows, err := s.DB.QueryContext(ctx, `
		SELECT c.id, c.body, c.viewpoint_id, c.topic_id, c.author_id,
		       c.created_at, c.updated_at, p.name as author_name
		FROM collab_comment c
		LEFT JOIN iam_profile p ON p.id = c.author_id
		WHERE c.topic_id = $1
		ORDER BY c.created_at ASC`, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.Body, &c.ViewpointID, &c.TopicID,
			&c.AuthorID, &c.CreatedAt, &c.UpdatedAt, &c.AuthorName); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}

	if comments == nil {
		comments = []Comment{}
	}
	return comments, nil
}

func (s *Service) CreateComment(ctx context.Context, topicID, authorID string, req CreateCommentRequest) (*Comment, error) {
	id := uuid.New().String()
	now := time.Now().UTC()

	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO collab_comment (id, body, viewpoint_id, topic_id, author_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		id, req.Body, req.ViewpointID, topicID, authorID, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert comment: %w", err)
	}

	return &Comment{
		ID:        id,
		Body:      req.Body,
		TopicID:   topicID,
		AuthorID:  authorID,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

func (s *Service) DeleteComment(ctx context.Context, commentID string) error {
	_, err := s.DB.ExecContext(ctx, "DELETE FROM collab_comment WHERE id = $1", commentID)
	return err
}

// --- Viewpoints ---

func (s *Service) listViewpoints(ctx context.Context, topicID string, limit int) ([]Viewpoint, error) {
	query := `
		SELECT id, guid, topic_id, camera_type,
		       camera_position_x, camera_position_y, camera_position_z,
		       camera_direction_x, camera_direction_y, camera_direction_z,
		       camera_up_x, camera_up_y, camera_up_z,
		       camera_fov, camera_view_world_scale,
		       snapshot_data, components, clipping_planes, created_at
		FROM collab_viewpoint
		WHERE topic_id = $1
		ORDER BY created_at ASC`

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}

	rows, err := s.DB.QueryContext(ctx, query, topicID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var viewpoints []Viewpoint
	for rows.Next() {
		var v Viewpoint
		var snapshotData []byte

		err := rows.Scan(
			&v.ID, &v.GUID, &v.TopicID, &v.CameraType,
			&v.CameraPosition.X, &v.CameraPosition.Y, &v.CameraPosition.Z,
			&v.CameraDirection.X, &v.CameraDirection.Y, &v.CameraDirection.Z,
			&v.CameraUp.X, &v.CameraUp.Y, &v.CameraUp.Z,
			&v.FieldOfView, &v.ViewWorldScale,
			&snapshotData, &v.Components, &v.ClippingPlanes, &v.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		// Convert snapshot to base64 data URL if present
		if len(snapshotData) > 0 {
			encoded := "data:image/png;base64," + encodeBase64(snapshotData)
			v.SnapshotBase64 = &encoded
		}

		viewpoints = append(viewpoints, v)
	}

	if viewpoints == nil {
		viewpoints = []Viewpoint{}
	}
	return viewpoints, nil
}

func (s *Service) CreateViewpoint(ctx context.Context, topicID string, req CreateViewpointRequest) (*Viewpoint, error) {
	id := uuid.New().String()
	guid := uuid.New().String()
	now := time.Now().UTC()

	// Decode base64 snapshot if provided
	var snapshotData []byte
	if req.SnapshotBase64 != nil {
		snapshotData = decodeBase64DataURL(*req.SnapshotBase64)
	}

	componentsJSON, _ := json.Marshal(req.Components)
	clippingJSON, _ := json.Marshal(req.ClippingPlanes)

	_, err := s.DB.ExecContext(ctx, `
		INSERT INTO collab_viewpoint (id, guid, topic_id, camera_type,
		    camera_position_x, camera_position_y, camera_position_z,
		    camera_direction_x, camera_direction_y, camera_direction_z,
		    camera_up_x, camera_up_y, camera_up_z,
		    camera_fov, camera_view_world_scale,
		    snapshot_data, components, clipping_planes, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)`,
		id, guid, topicID, req.CameraType,
		req.CameraPosition.X, req.CameraPosition.Y, req.CameraPosition.Z,
		req.CameraDirection.X, req.CameraDirection.Y, req.CameraDirection.Z,
		req.CameraUp.X, req.CameraUp.Y, req.CameraUp.Z,
		req.FieldOfView, req.ViewWorldScale,
		snapshotData, componentsJSON, clippingJSON, now,
	)
	if err != nil {
		return nil, fmt.Errorf("insert viewpoint: %w", err)
	}

	return &Viewpoint{
		ID:              id,
		GUID:            guid,
		TopicID:         topicID,
		CameraType:      req.CameraType,
		CameraPosition:  req.CameraPosition,
		CameraDirection: req.CameraDirection,
		CameraUp:        req.CameraUp,
		FieldOfView:     req.FieldOfView,
		ViewWorldScale:  req.ViewWorldScale,
		CreatedAt:       now,
	}, nil
}

func (s *Service) GetSnapshot(ctx context.Context, viewpointID string) ([]byte, string, error) {
	var data []byte
	var snapType string
	err := s.DB.QueryRowContext(ctx,
		"SELECT snapshot_data, COALESCE(snapshot_type, 'png') FROM collab_viewpoint WHERE id = $1",
		viewpointID,
	).Scan(&data, &snapType)
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("no snapshot")
	}
	return data, "image/" + snapType, nil
}

// --- BCF Export/Import ---

func (s *Service) ExportBCF(ctx context.Context, projectID string) ([]byte, error) {
	topics, err := s.ListTopics(ctx, projectID, TopicFilters{})
	if err != nil {
		return nil, err
	}

	// Fetch full data for each topic
	var fullTopics []Topic
	for _, t := range topics {
		full, err := s.GetTopic(ctx, t.ID)
		if err != nil {
			continue
		}
		fullTopics = append(fullTopics, *full)
	}

	return ExportBCFZip(fullTopics)
}

func (s *Service) ImportBCF(ctx context.Context, projectID, importerID string, file io.Reader) (int, error) {
	data, err := io.ReadAll(file)
	if err != nil {
		return 0, fmt.Errorf("read file: %w", err)
	}

	topics, err := ParseBCFZip(data)
	if err != nil {
		return 0, fmt.Errorf("parse BCF: %w", err)
	}

	count := 0
	for _, imported := range topics {
		req := CreateTopicRequest{
			Title:       imported.Title,
			Description: imported.Description,
			Priority:    imported.Priority,
			TopicType:   imported.TopicType,
		}

		if len(imported.Viewpoints) > 0 {
			vp := imported.Viewpoints[0]
			req.Viewpoint = &CreateViewpointRequest{
				CameraType:      vp.CameraType,
				CameraPosition:  vp.CameraPosition,
				CameraDirection: vp.CameraDirection,
				CameraUp:        vp.CameraUp,
				FieldOfView:     vp.FieldOfView,
				ViewWorldScale:  vp.ViewWorldScale,
				SnapshotBase64:  vp.SnapshotBase64,
				Components:      vp.Components,
				ClippingPlanes:  vp.ClippingPlanes,
			}
		}

		_, err := s.CreateTopic(ctx, projectID, importerID, req)
		if err != nil {
			continue
		}
		count++
	}

	return count, nil
}
