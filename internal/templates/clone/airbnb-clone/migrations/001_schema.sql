-- airbnb-clone schema
-- Tables: np_listings, np_bookings, np_reviews, np_users (extends auth)

CREATE TABLE IF NOT EXISTS np_listings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    host_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    title TEXT NOT NULL,
    description TEXT,
    location TEXT NOT NULL,
    price_per_night NUMERIC(10,2) NOT NULL,
    max_guests INTEGER NOT NULL DEFAULT 1,
    photos JSONB DEFAULT '[]',
    amenities JSONB DEFAULT '[]',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_bookings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    listing_id UUID NOT NULL REFERENCES np_listings(id) ON DELETE CASCADE,
    guest_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    check_in DATE NOT NULL,
    check_out DATE NOT NULL,
    total_price NUMERIC(10,2) NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending','confirmed','cancelled','completed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS np_reviews (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    booking_id UUID NOT NULL REFERENCES np_bookings(id) ON DELETE CASCADE,
    reviewer_id UUID NOT NULL REFERENCES auth.users(id) ON DELETE CASCADE,
    listing_id UUID NOT NULL REFERENCES np_listings(id) ON DELETE CASCADE,
    rating INTEGER NOT NULL CHECK (rating BETWEEN 1 AND 5),
    comment TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Enable RLS
ALTER TABLE np_listings ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_bookings ENABLE ROW LEVEL SECURITY;
ALTER TABLE np_reviews ENABLE ROW LEVEL SECURITY;

-- Listings: hosts own their listings; everyone can read active ones
CREATE POLICY np_listings_select ON np_listings FOR SELECT USING (is_active = TRUE);
CREATE POLICY np_listings_host_all ON np_listings USING (host_id = auth.uid());

-- Bookings: guests see their own; hosts see bookings for their listings
CREATE POLICY np_bookings_guest ON np_bookings USING (guest_id = auth.uid());
CREATE POLICY np_bookings_host ON np_bookings USING (
    listing_id IN (SELECT id FROM np_listings WHERE host_id = auth.uid())
);

-- Reviews: everyone reads; reviewer owns their review
CREATE POLICY np_reviews_select ON np_reviews FOR SELECT USING (TRUE);
CREATE POLICY np_reviews_owner ON np_reviews USING (reviewer_id = auth.uid());

-- Indexes
CREATE INDEX IF NOT EXISTS np_listings_host_idx ON np_listings(host_id);
CREATE INDEX IF NOT EXISTS np_bookings_listing_idx ON np_bookings(listing_id);
CREATE INDEX IF NOT EXISTS np_bookings_guest_idx ON np_bookings(guest_id);
CREATE INDEX IF NOT EXISTS np_reviews_listing_idx ON np_reviews(listing_id);
