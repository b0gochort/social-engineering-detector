-- Create access_requests table
CREATE TABLE access_requests (
    id SERIAL PRIMARY KEY,
    incident_id INTEGER NOT NULL REFERENCES incidents(id) ON DELETE CASCADE,
    parent_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    child_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, approved, rejected
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    responded_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for faster lookups
CREATE INDEX idx_access_requests_incident_id ON access_requests(incident_id);
CREATE INDEX idx_access_requests_parent_id ON access_requests(parent_id);
CREATE INDEX idx_access_requests_child_id ON access_requests(child_id);
CREATE INDEX idx_access_requests_status ON access_requests(status);

-- Add constraint to ensure status is valid
ALTER TABLE access_requests ADD CONSTRAINT check_status
    CHECK (status IN ('pending', 'approved', 'rejected'));
