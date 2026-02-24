use std::collections::HashMap;

/// Find the longest of two string slices.
pub fn longest<'a>(x: &'a str, y: &'a str) -> &'a str {
    if x.len() > y.len() { x } else { y }
}

/// Map items from one type to another.
pub fn map_items<T, U>(items: Vec<T>, f: fn(T) -> U) -> Vec<U> {
    items.into_iter().map(f).collect()
}

/// Print all items that implement Display and Debug.
pub fn print_all<T>(items: &[T]) where T: std::fmt::Display + std::fmt::Debug {
    for item in items {
        println!("{:?}", item);
    }
}

/// A generic cache with hash and equality constraints.
pub struct Cache<K: Hash + Eq, V> {
    data: HashMap<K, V>,
    capacity: usize,
}

/// A reference wrapper with a lifetime.
pub struct Ref<'a, T> {
    inner: &'a T,
}

impl<K: Hash + Eq, V> Cache<K, V> {
    pub fn new(capacity: usize) -> Self {
        Cache {
            data: HashMap::with_capacity(capacity),
            capacity,
        }
    }

    pub fn get(&self, key: &K) -> Option<&V> {
        self.data.get(key)
    }

    pub fn insert(&mut self, key: K, value: V) -> Option<V> {
        if self.data.len() >= self.capacity {
            return None;
        }
        self.data.insert(key, value)
    }
}

impl<'a, T> Ref<'a, T> {
    pub fn new(inner: &'a T) -> Self {
        Ref { inner }
    }

    pub fn get(&self) -> &T {
        self.inner
    }
}

/// Convert with a where clause and multiple bounds.
pub fn convert<T, U>(input: T) -> U where T: Into<U> + Clone, U: Default {
    input.into()
}