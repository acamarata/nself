-- airbnb-clone demo data (skipped with --no-seed)
-- Requires auth users to exist; uses placeholder UUIDs for demo.

INSERT INTO np_listings (id, host_id, title, description, location, price_per_night, max_guests, amenities)
VALUES
    ('00000000-0000-0000-0000-000000000001',
     '00000000-0000-0000-0000-000000000010',
     'Cozy City Apartment',
     'A bright 1-bedroom apartment in the city center.',
     'San Francisco, CA',
     120.00, 2,
     '["WiFi","Kitchen","Air Conditioning"]'),
    ('00000000-0000-0000-0000-000000000002',
     '00000000-0000-0000-0000-000000000010',
     'Beach House with Ocean View',
     'Spacious 3-bedroom beach house steps from the water.',
     'Malibu, CA',
     350.00, 8,
     '["WiFi","Pool","Beach Access","BBQ"]')
ON CONFLICT (id) DO NOTHING;
