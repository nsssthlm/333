-- Migration 003: Add Speckle file-to-model mapping table
-- Tracks which ValvX files have been imported into Speckle and their processing status

BEGIN;

CREATE TABLE public.arca_speckle_mapping (
    file_version_id uuid NOT NULL,
    speckle_model_id text NOT NULL,
    speckle_version_id text,
    speckle_object_id text,
    status text NOT NULL DEFAULT 'pending',
    error_message text,
    created_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    updated_at timestamp without time zone DEFAULT CURRENT_TIMESTAMP NOT NULL,
    PRIMARY KEY (file_version_id),
    CONSTRAINT fk_speckle_mapping_file_version FOREIGN KEY (file_version_id) REFERENCES public.arca_file_version(id)
);

CREATE INDEX idx_speckle_mapping_status ON public.arca_speckle_mapping(status);
CREATE INDEX idx_speckle_mapping_model ON public.arca_speckle_mapping(speckle_model_id);

-- Update migration version
UPDATE public.migration_version SET version = 3;

COMMIT;
