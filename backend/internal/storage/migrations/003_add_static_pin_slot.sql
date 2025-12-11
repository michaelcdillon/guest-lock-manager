-- Add slot_number to static_pins to persist desired slot per PIN
ALTER TABLE static_pins ADD COLUMN slot_number INTEGER NOT NULL DEFAULT 1;

-- Backfill existing rows to default slot 1
UPDATE static_pins SET slot_number = 1 WHERE slot_number IS NULL;


