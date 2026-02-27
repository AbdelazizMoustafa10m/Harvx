/// A public function.
pub fn public_func() -> u32 {
    42
}

/// A crate-visible function.
pub(crate) fn crate_func() -> u32 {
    42
}

/// A parent-module-visible function.
pub(super) fn super_func() -> u32 {
    42
}

/// A private function.
fn private_func() -> u32 {
    42
}

/// A public constant.
pub const PUBLIC_CONST: u32 = 100;

/// A crate-visible constant.
pub(crate) const CRATE_CONST: &str = "hello";

/// A private constant.
const PRIVATE_CONST: bool = true;

/// A public struct with mixed field visibility.
pub struct MixedVisibility {
    pub public_field: String,
    pub(crate) crate_field: u32,
    pub(super) super_field: bool,
    private_field: Vec<u8>,
}

/// A crate-visible struct.
pub(crate) struct CrateStruct {
    value: i32,
}

/// A public static.
pub static GLOBAL: AtomicUsize = AtomicUsize::new(0);

/// A crate-visible type alias.
pub(crate) type InternalId = u64;

pub(crate) use crate::internal::Helper;
pub use crate::config::Config;