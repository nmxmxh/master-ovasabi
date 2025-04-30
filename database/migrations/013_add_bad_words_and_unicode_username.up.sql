-- Modify username column to support Unicode characters
ALTER TABLE service_user ALTER COLUMN username TYPE VARCHAR(64) COLLATE "und-x-icu";
DROP INDEX IF EXISTS idx_service_user_username;
CREATE INDEX idx_service_user_username ON service_user(username COLLATE "und-x-icu");

-- Create bad words table with language support
CREATE TABLE bad_words (
    id SERIAL PRIMARY KEY,
    word VARCHAR(64) NOT NULL,
    language VARCHAR(5) NOT NULL, -- ISO 639-1 language code + optional region
    category VARCHAR(32) NOT NULL, -- e.g., profanity, slur, offensive, etc.
    severity INTEGER NOT NULL, -- 1: mild, 2: moderate, 3: severe
    is_regex BOOLEAN DEFAULT false, -- whether the word is a regex pattern
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create unique constraint on word + language
CREATE UNIQUE INDEX idx_bad_words_word_lang ON bad_words(word, language);
CREATE INDEX idx_bad_words_language ON bad_words(language);
CREATE INDEX idx_bad_words_category ON bad_words(category);

-- Create function to normalize username
CREATE OR REPLACE FUNCTION normalize_username(username TEXT)
RETURNS TEXT AS $$
BEGIN
    -- Convert to NFKC normalized form (Compatibility Decomposition, followed by Composition)
    -- This handles various Unicode equivalences
    RETURN normalize(lower(username), NFKC);
END;
$$ LANGUAGE plpgsql IMMUTABLE STRICT;

-- Create function to check for bad words
CREATE OR REPLACE FUNCTION contains_bad_word(input_text TEXT, input_lang TEXT DEFAULT 'en')
RETURNS BOOLEAN AS $$
DECLARE
    bad_word RECORD;
BEGIN
    -- First check exact matches
    FOR bad_word IN 
        SELECT word, is_regex 
        FROM bad_words 
        WHERE language = input_lang 
        OR language = '*'  -- Universal bad words
    LOOP
        IF bad_word.is_regex THEN
            IF input_text ~ bad_word.word THEN
                RETURN TRUE;
            END IF;
        ELSE
            IF position(lower(bad_word.word) in lower(input_text)) > 0 THEN
                RETURN TRUE;
            END IF;
        END IF;
    END LOOP;
    
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Add trigger to validate username
CREATE OR REPLACE FUNCTION validate_username()
RETURNS TRIGGER AS $$
DECLARE
    normalized_username TEXT;
BEGIN
    -- Normalize the username
    normalized_username := normalize_username(NEW.username);
    
    -- Check length of normalized username (in characters, not bytes)
    IF length(normalized_username) < 3 OR length(normalized_username) > 64 THEN
        RAISE EXCEPTION 'Username must be between 3 and 64 characters';
    END IF;
    
    -- Check for bad words in multiple languages
    IF contains_bad_word(normalized_username, 'en') OR 
       contains_bad_word(normalized_username, 'es') OR
       contains_bad_word(normalized_username, 'fr') OR
       contains_bad_word(normalized_username, 'de') OR
       contains_bad_word(normalized_username, 'it') OR
       contains_bad_word(normalized_username, 'pt') OR
       contains_bad_word(normalized_username, 'ru') OR
       contains_bad_word(normalized_username, 'zh') OR
       contains_bad_word(normalized_username, 'ja') OR
       contains_bad_word(normalized_username, 'ko') OR
       contains_bad_word(normalized_username, '*') THEN
        RAISE EXCEPTION 'Username contains inappropriate content';
    END IF;
    
    -- Store normalized username
    NEW.username := normalized_username;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for username validation
CREATE TRIGGER validate_username_trigger
    BEFORE INSERT OR UPDATE ON service_user
    FOR EACH ROW
    EXECUTE FUNCTION validate_username();

-- Insert some example bad words (in practice, you'd want a more comprehensive list)
INSERT INTO bad_words (word, language, category, severity) VALUES
    -- English
    ('fuck\w*', 'en', 'profanity', 2, true),
    ('shit\w*', 'en', 'profanity', 2, true),
    ('ass(?:hole)?', 'en', 'profanity', 2, true),
    -- Spanish
    ('puta', 'es', 'profanity', 2, false),
    ('mierda', 'es', 'profanity', 2, false),
    -- French
    ('merde', 'fr', 'profanity', 2, false),
    ('putain', 'fr', 'profanity', 2, false),
    -- German
    ('scheiße', 'de', 'profanity', 2, false),
    ('arschloch', 'de', 'profanity', 2, false),
    -- Universal patterns (work across languages)
    ('\b(admin|mod|staff)\d+\b', '*', 'impersonation', 3, true),
    ('^\d+$', '*', 'spam', 1, true),  -- username can't be all numbers
    ('^[._-]', '*', 'formatting', 1, true),  -- can't start with punctuation
    ('[._-]$', '*', 'formatting', 1, true);  -- can't end with punctuation

-- Create trigger for bad_words updated_at
CREATE TRIGGER update_bad_words_updated_at
    BEFORE UPDATE ON bad_words
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column(); 