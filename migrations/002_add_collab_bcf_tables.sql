-- Migration 002: Add BCF (BIM Collaboration Format) tables
-- Module prefix: collab_ (collaboration)
-- Follows existing ValvX naming conventions from migration 001

BEGIN;

-- BCF Topics
CREATE TABLE public.collab_topic (
    id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    guid text NOT NULL UNIQUE,
    title text NOT NULL,
    description text,
    priority text,
    topic_type text,
    topic_status text DEFAULT 'Open' NOT NULL,
    stage text,
    assigned_to uuid,
    due_date date,
    labels text[],
    project_id uuid NOT NULL,
    creator_id uuid NOT NULL,
    modified_by uuid,
    PRIMARY KEY (id),
    CONSTRAINT fk_collab_topic_project FOREIGN KEY (project_id) REFERENCES public.core_project(id),
    CONSTRAINT fk_collab_topic_creator FOREIGN KEY (creator_id) REFERENCES public.iam_profile(id),
    CONSTRAINT fk_collab_topic_assigned FOREIGN KEY (assigned_to) REFERENCES public.iam_profile(id),
    CONSTRAINT fk_collab_topic_modified FOREIGN KEY (modified_by) REFERENCES public.iam_profile(id)
);

CREATE INDEX idx_collab_topic_project ON public.collab_topic(project_id);
CREATE INDEX idx_collab_topic_status ON public.collab_topic(topic_status);
CREATE INDEX idx_collab_topic_guid ON public.collab_topic(guid);

-- BCF Viewpoints (camera state + component visibility + snapshots)
CREATE TABLE public.collab_viewpoint (
    id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    guid text NOT NULL UNIQUE,
    topic_id uuid NOT NULL,
    camera_type text NOT NULL,
    camera_position_x double precision NOT NULL,
    camera_position_y double precision NOT NULL,
    camera_position_z double precision NOT NULL,
    camera_direction_x double precision NOT NULL,
    camera_direction_y double precision NOT NULL,
    camera_direction_z double precision NOT NULL,
    camera_up_x double precision NOT NULL,
    camera_up_y double precision NOT NULL,
    camera_up_z double precision NOT NULL,
    camera_fov double precision,
    camera_view_world_scale double precision,
    snapshot_data bytea,
    snapshot_type text DEFAULT 'png',
    components jsonb,
    clipping_planes jsonb,
    lines jsonb,
    PRIMARY KEY (id),
    CONSTRAINT fk_collab_viewpoint_topic FOREIGN KEY (topic_id) REFERENCES public.collab_topic(id) ON DELETE CASCADE
);

CREATE INDEX idx_collab_viewpoint_topic ON public.collab_viewpoint(topic_id);

-- BCF Comments
CREATE TABLE public.collab_comment (
    id uuid NOT NULL,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    body text NOT NULL,
    viewpoint_id uuid,
    topic_id uuid NOT NULL,
    author_id uuid NOT NULL,
    PRIMARY KEY (id),
    CONSTRAINT fk_collab_comment_topic FOREIGN KEY (topic_id) REFERENCES public.collab_topic(id) ON DELETE CASCADE,
    CONSTRAINT fk_collab_comment_viewpoint FOREIGN KEY (viewpoint_id) REFERENCES public.collab_viewpoint(id) ON DELETE SET NULL,
    CONSTRAINT fk_collab_comment_author FOREIGN KEY (author_id) REFERENCES public.iam_profile(id)
);

CREATE INDEX idx_collab_comment_topic ON public.collab_comment(topic_id);

-- BCF Topic <-> File Version link
CREATE TABLE public.collab_topic_file (
    topic_id uuid NOT NULL,
    file_version_id uuid NOT NULL,
    PRIMARY KEY (topic_id, file_version_id),
    CONSTRAINT fk_collab_topic_file_topic FOREIGN KEY (topic_id) REFERENCES public.collab_topic(id) ON DELETE CASCADE,
    CONSTRAINT fk_collab_topic_file_version FOREIGN KEY (file_version_id) REFERENCES public.arca_file_version(id)
);

-- Update migration version
UPDATE public.migration_version SET version = 2;

COMMIT;
