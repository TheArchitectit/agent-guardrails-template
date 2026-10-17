-- Migration: Add webhook signing-key rotation columns
-- Version: 019
--
-- Spec 17 R17-10 requires webhook material to rotate with a staged replacement
-- and a bounded overlap: the outgoing signing key stays valid for a limited
-- window while a receiver cuts over, and the rotation is recorded. Before this
-- migration a webhook could only be replaced in place, with no previous-key
-- window and no record of when the key changed.

ALTER TABLE webhook_configs
    ADD COLUMN IF NOT EXISTS secret_hmac_previous TEXT,
    ADD COLUMN IF NOT EXISTS secret_hmac_rotated_at TIMESTAMP WITH TIME ZONE;

COMMENT ON COLUMN webhook_configs.secret_hmac_previous IS
    'Outgoing webhook signing key kept valid for the bounded rotation overlap (Spec 17 R17-10); never returned on any read path';
COMMENT ON COLUMN webhook_configs.secret_hmac_rotated_at IS
    'When the staged signing-key replacement was applied; the previous key is offered only inside the bounded overlap (Spec 17 R17-10)';
