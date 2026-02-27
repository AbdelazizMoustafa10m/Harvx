use std::collections::HashMap;
use std::fmt;

/// A cache with time-based expiration for stored values.
#[derive(Debug)]
pub struct Cache<K, V> {
    entries: HashMap<K, CacheEntry<V>>,
    ttl: std::time::Duration,
    max_size: usize,
}

struct CacheEntry<V> {
    value: V,
    expires_at: std::time::Instant,
}

impl<K: Eq + std::hash::Hash, V: Clone> Cache<K, V> {
    /// Creates a new cache with the given TTL and maximum size.
    pub fn new(ttl: std::time::Duration, max_size: usize) -> Self {
        Self {
            entries: HashMap::new(),
            ttl,
            max_size,
        }
    }

    /// Retrieves a value from the cache if it exists and hasn't expired.
    pub fn get(&self, key: &K) -> Option<&V> {
        self.entries.get(key).and_then(|entry| {
            if entry.expires_at > std::time::Instant::now() {
                Some(&entry.value)
            } else {
                None
            }
        })
    }

    /// Inserts a value into the cache with the configured TTL.
    pub fn insert(&mut self, key: K, value: V) {
        if self.entries.len() >= self.max_size {
            self.evict_expired();
        }
        self.entries.insert(key, CacheEntry {
            value,
            expires_at: std::time::Instant::now() + self.ttl,
        });
    }

    fn evict_expired(&mut self) {
        let now = std::time::Instant::now();
        self.entries.retain(|_, entry| entry.expires_at > now);
    }
}

impl<K, V: fmt::Display> fmt::Display for Cache<K, V> {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "Cache(entries={}, ttl={:?})", self.entries.len(), self.ttl)
    }
}

pub const DEFAULT_TTL_SECS: u64 = 300;
pub const DEFAULT_MAX_SIZE: usize = 1000;
