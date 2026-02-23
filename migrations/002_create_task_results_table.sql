CREATE TABLE IF NOT EXISTS task_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    run_at TIMESTAMP WITH TIME ZONE NOT NULL,
    status_code INTEGER NOT NULL,
    success BOOLEAN NOT NULL,
    response_headers JSONB,
    response_body TEXT,
    error_message TEXT,
    duration_ms BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_task_id FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
);

CREATE INDEX idx_task_results_task_id ON task_results(task_id);
CREATE INDEX idx_task_results_run_at ON task_results(run_at DESC);
CREATE INDEX idx_task_results_success ON task_results(success);
CREATE INDEX idx_task_results_created_at ON task_results(created_at DESC);
