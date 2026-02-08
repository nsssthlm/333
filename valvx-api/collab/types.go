package collab

import (
	"encoding/json"
	"time"
)

// Topic represents a BCF topic (issue/request).
type Topic struct {
	ID             string      `json:"id"`
	GUID           string      `json:"guid"`
	Title          string      `json:"title"`
	Description    *string     `json:"description,omitempty"`
	Priority       *string     `json:"priority,omitempty"`
	TopicType      *string     `json:"topicType,omitempty"`
	TopicStatus    string      `json:"topicStatus"`
	Stage          *string     `json:"stage,omitempty"`
	AssignedTo     *string     `json:"assignedTo,omitempty"`
	AssignedToName *string     `json:"assignedToName,omitempty"`
	DueDate        *string     `json:"dueDate,omitempty"`
	Labels         []string    `json:"labels,omitempty"`
	ProjectID      string      `json:"projectId"`
	CreatorID      string      `json:"creatorId"`
	CreatorName    *string     `json:"creatorName,omitempty"`
	ModifiedBy     *string     `json:"modifiedBy,omitempty"`
	Viewpoints     []Viewpoint `json:"viewpoints,omitempty"`
	Comments       []Comment   `json:"comments,omitempty"`
	FileVersionIDs []string    `json:"fileVersionIds,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
	UpdatedAt      time.Time   `json:"updatedAt"`
}

// Comment represents a BCF comment on a topic.
type Comment struct {
	ID          string    `json:"id"`
	Body        string    `json:"body"`
	ViewpointID *string   `json:"viewpointId,omitempty"`
	TopicID     string    `json:"topicId"`
	AuthorID    string    `json:"authorId"`
	AuthorName  *string   `json:"authorName,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// Viewpoint represents a BCF viewpoint (camera state + component visibility).
type Viewpoint struct {
	ID              string           `json:"id"`
	GUID            string           `json:"guid"`
	TopicID         string           `json:"topicId"`
	CameraType      string           `json:"cameraType"`
	CameraPosition  Vector3          `json:"cameraPosition"`
	CameraDirection Vector3          `json:"cameraDirection"`
	CameraUp        Vector3          `json:"cameraUp"`
	FieldOfView     *float64         `json:"fieldOfView,omitempty"`
	ViewWorldScale  *float64         `json:"viewWorldScale,omitempty"`
	SnapshotBase64  *string          `json:"snapshotBase64,omitempty"`
	Components      *json.RawMessage `json:"components,omitempty"`
	ClippingPlanes  *json.RawMessage `json:"clippingPlanes,omitempty"`
	Lines           *json.RawMessage `json:"lines,omitempty"`
	CreatedAt       time.Time        `json:"createdAt"`
}

// Vector3 is a 3D coordinate.
type Vector3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// TopicFilters for listing topics.
type TopicFilters struct {
	Status     string
	Priority   string
	AssignedTo string
}

// CreateTopicRequest is the request body for creating a topic.
type CreateTopicRequest struct {
	Title          string   `json:"title"`
	Description    *string  `json:"description,omitempty"`
	Priority       *string  `json:"priority,omitempty"`
	TopicType      *string  `json:"topicType,omitempty"`
	AssignedTo     *string  `json:"assignedTo,omitempty"`
	DueDate        *string  `json:"dueDate,omitempty"`
	Labels         []string `json:"labels,omitempty"`
	FileVersionIDs []string `json:"fileVersionIds,omitempty"`
	Viewpoint      *CreateViewpointRequest `json:"viewpoint,omitempty"`
}

// CreateCommentRequest is the request body for creating a comment.
type CreateCommentRequest struct {
	Body        string  `json:"body"`
	ViewpointID *string `json:"viewpointId,omitempty"`
}

// CreateViewpointRequest is the request body for creating a viewpoint.
type CreateViewpointRequest struct {
	CameraType      string           `json:"cameraType"`
	CameraPosition  Vector3          `json:"cameraPosition"`
	CameraDirection Vector3          `json:"cameraDirection"`
	CameraUp        Vector3          `json:"cameraUp"`
	FieldOfView     *float64         `json:"fieldOfView,omitempty"`
	ViewWorldScale  *float64         `json:"viewWorldScale,omitempty"`
	SnapshotBase64  *string          `json:"snapshotBase64,omitempty"`
	Components      *json.RawMessage `json:"components,omitempty"`
	ClippingPlanes  *json.RawMessage `json:"clippingPlanes,omitempty"`
}
