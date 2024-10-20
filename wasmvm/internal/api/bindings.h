/* Licensed under Apache-2.0. Copyright see https://github.com/CosmWasm/wasmvm/blob/main/NOTICE. */

/* Generated with cbindgen:0.27.0 */

/* Warning, this file is autogenerated by cbindgen. Don't modify this manually. */

#include <stdarg.h>
#include <stdbool.h>
#include <stdint.h>
#include <stdlib.h>

enum ErrnoValue {
  ErrnoValue_Success = 0,
  ErrnoValue_Other = 1,
  ErrnoValue_OutOfGas = 2,
};
typedef int32_t ErrnoValue;

/**
 * This enum gives names to the status codes returned from Go callbacks to Rust.
 * The Go code will return one of these variants when returning.
 *
 * 0 means no error, all the other cases are some sort of error.
 *
 */
enum GoError {
  GoError_None = 0,
  /**
   * Go panicked for an unexpected reason.
   */
  GoError_Panic = 1,
  /**
   * Go received a bad argument from Rust
   */
  GoError_BadArgument = 2,
  /**
   * Ran out of gas while using the SDK (e.g. storage). This can come from the Cosmos SDK gas meter
   * (https://github.com/cosmos/cosmos-sdk/blob/v0.45.4/store/types/gas.go#L29-L32).
   */
  GoError_OutOfGas = 3,
  /**
   * Error while trying to serialize data in Go code (typically json.Marshal)
   */
  GoError_CannotSerialize = 4,
  /**
   * An error happened during normal operation of a Go callback, which should be fed back to the contract
   */
  GoError_User = 5,
  /**
   * An error type that should never be created by us. It only serves as a fallback for the i32 to GoError conversion.
   */
  GoError_Other = -1,
};
typedef int32_t GoError;

typedef struct cache_t {

} cache_t;

/**
 * A view into an externally owned byte slice (Go `[]byte`).
 * Use this for the current call only. A view cannot be copied for safety reasons.
 * If you need a copy, use [`ByteSliceView::to_owned`].
 *
 * Go's nil value is fully supported, such that we can differentiate between nil and an empty slice.
 */
typedef struct ByteSliceView {
  /**
   * True if and only if the byte slice is nil in Go. If this is true, the other fields must be ignored.
   */
  bool is_nil;
  const uint8_t *ptr;
  uintptr_t len;
} ByteSliceView;

/**
 * An optional Vector type that requires explicit creation and destruction
 * and can be sent via FFI.
 * It can be created from `Option<Vec<u8>>` and be converted into `Option<Vec<u8>>`.
 *
 * This type is always created in Rust and always dropped in Rust.
 * If Go code want to create it, it must instruct Rust to do so via the
 * [`new_unmanaged_vector`] FFI export. If Go code wants to consume its data,
 * it must create a copy and instruct Rust to destroy it via the
 * [`destroy_unmanaged_vector`] FFI export.
 *
 * An UnmanagedVector is immutable.
 *
 * ## Ownership
 *
 * Ownership is the right and the obligation to destroy an `UnmanagedVector`
 * exactly once. Both Rust and Go can create an `UnmanagedVector`, which gives
 * then ownership. Sometimes it is necessary to transfer ownership.
 *
 * ### Transfer ownership from Rust to Go
 *
 * When an `UnmanagedVector` was created in Rust using [`UnmanagedVector::new`], [`UnmanagedVector::default`]
 * or [`new_unmanaged_vector`], it can be passed to Go as a return value (see e.g. [load_wasm][crate::load_wasm]).
 * Rust then has no chance to destroy the vector anymore, so ownership is transferred to Go.
 * In Go, the data has to be copied to a garbage collected `[]byte`. Then the vector must be destroyed
 * using [`destroy_unmanaged_vector`].
 *
 * ### Transfer ownership from Go to Rust
 *
 * When Rust code calls into Go (using the vtable methods), return data or error messages must be created
 * in Go. This is done by calling [`new_unmanaged_vector`] from Go, which copies data into a newly created
 * `UnmanagedVector`. Since Go created it, it owns it. The ownership is then passed to Rust via the
 * mutable return value pointers. On the Rust side, the vector is destroyed using [`UnmanagedVector::consume`].
 *
 * ## Examples
 *
 * Transferring ownership from Rust to Go using return values of FFI calls:
 *
 * ```
 * # use wasmvm::{cache_t, ByteSliceView, UnmanagedVector};
 * #[no_mangle]
 * pub extern "C" fn save_wasm_to_cache(
 *     cache: *mut cache_t,
 *     wasm: ByteSliceView,
 *     error_msg: Option<&mut UnmanagedVector>,
 * ) -> UnmanagedVector {
 *     # let checksum: Vec<u8> = Default::default();
 *     // some operation producing a `let checksum: Vec<u8>`
 *
 *     UnmanagedVector::new(Some(checksum)) // this unmanaged vector is owned by the caller
 * }
 * ```
 *
 * Transferring ownership from Go to Rust using return value pointers:
 *
 * ```rust
 * # use cosmwasm_vm::{BackendResult, GasInfo};
 * # use wasmvm::{Db, GoError, U8SliceView, UnmanagedVector};
 * fn db_read(db: &Db, key: &[u8]) -> BackendResult<Option<Vec<u8>>> {
 *
 *     // Create a None vector in order to reserve memory for the result
 *     let mut output = UnmanagedVector::default();
 *
 *     // …
 *     # let mut error_msg = UnmanagedVector::default();
 *     # let mut used_gas = 0_u64;
 *     # let read_db = db.vtable.read_db.unwrap();
 *
 *     let go_error: GoError = read_db(
 *         db.state,
 *         db.gas_meter,
 *         &mut used_gas as *mut u64,
 *         U8SliceView::new(Some(key)),
 *         // Go will create a new UnmanagedVector and override this address
 *         &mut output as *mut UnmanagedVector,
 *         &mut error_msg as *mut UnmanagedVector,
 *     )
 *     .into();
 *
 *     // We now own the new UnmanagedVector written to the pointer and must destroy it
 *     let value = output.consume();
 *
 *     // Some gas processing and error handling
 *     # let gas_info = GasInfo::free();
 *
 *     (Ok(value), gas_info)
 * }
 * ```
 *
 *
 * If you want to mutate data, you need to comsume the vector and create a new one:
 *
 * ```rust
 * # use wasmvm::{UnmanagedVector};
 * # let input = UnmanagedVector::new(Some(vec![0xAA]));
 * let mut mutable: Vec<u8> = input.consume().unwrap_or_default();
 * assert_eq!(mutable, vec![0xAA]);
 *
 * // `input` is now gone and we cam do everything we want to `mutable`,
 * // including operations that reallocate the underlying data.
 *
 * mutable.push(0xBB);
 * mutable.push(0xCC);
 *
 * assert_eq!(mutable, vec![0xAA, 0xBB, 0xCC]);
 *
 * let output = UnmanagedVector::new(Some(mutable));
 *
 * // `output` is ready to be passed around
 * ```
 */
typedef struct UnmanagedVector {
  /**
   * True if and only if this is None. If this is true, the other fields must be ignored.
   */
  bool is_none;
  uint8_t *ptr;
  uintptr_t len;
  uintptr_t cap;
} UnmanagedVector;

/**
 * A version of `Option<u64>` that can be used safely in FFI.
 */
typedef struct OptionalU64 {
  bool is_some;
  uint64_t value;
} OptionalU64;

/**
 * The result type of the FFI function analyze_code.
 *
 * Please note that the unmanaged vector in `required_capabilities`
 * has to be destroyed exactly once. When calling `analyze_code`
 * from Go this is done via `C.destroy_unmanaged_vector`.
 */
typedef struct AnalysisReport {
  /**
   * `true` if and only if all required ibc exports exist as exported functions.
   * This does not guarantee they are functional or even have the correct signatures.
   */
  bool has_ibc_entry_points;
  /**
   * A UTF-8 encoded comma separated list of all entrypoints that
   * are exported by the contract.
   */
  struct UnmanagedVector entrypoints;
  /**
   * An UTF-8 encoded comma separated list of required capabilities.
   * This is never None/nil.
   */
  struct UnmanagedVector required_capabilities;
  /**
   * The migrate version of the contract.
   * This is None if the contract does not have a migrate version and the `migrate` entrypoint
   * needs to be called for every migration (if present).
   * If it is `Some(version)`, it only needs to be called if the `version` increased.
   */
  struct OptionalU64 contract_migrate_version;
} AnalysisReport;

typedef struct Metrics {
  uint32_t hits_pinned_memory_cache;
  uint32_t hits_memory_cache;
  uint32_t hits_fs_cache;
  uint32_t misses;
  uint64_t elements_pinned_memory_cache;
  uint64_t elements_memory_cache;
  uint64_t size_pinned_memory_cache;
  uint64_t size_memory_cache;
} Metrics;

/**
 * An opaque type. `*gas_meter_t` represents a pointer to Go memory holding the gas meter.
 */
typedef struct gas_meter_t {
  uint8_t _private[0];
} gas_meter_t;

typedef struct db_t {
  uint8_t _private[0];
} db_t;

/**
 * A view into a `Option<&[u8]>`, created and maintained by Rust.
 *
 * This can be copied into a []byte in Go.
 */
typedef struct U8SliceView {
  /**
   * True if and only if this is None. If this is true, the other fields must be ignored.
   */
  bool is_none;
  const uint8_t *ptr;
  uintptr_t len;
} U8SliceView;

/**
 * A reference to some tables on the Go side which allow accessing
 * the actual iterator instance.
 */
typedef struct IteratorReference {
  /**
   * An ID assigned to this contract call
   */
  uint64_t call_id;
  /**
   * An ID assigned to this iterator
   */
  uint64_t iterator_id;
} IteratorReference;

typedef struct IteratorVtable {
  int32_t (*next)(struct IteratorReference iterator,
                  struct gas_meter_t *gas_meter,
                  uint64_t *gas_used,
                  struct UnmanagedVector *key_out,
                  struct UnmanagedVector *value_out,
                  struct UnmanagedVector *err_msg_out);
  int32_t (*next_key)(struct IteratorReference iterator,
                      struct gas_meter_t *gas_meter,
                      uint64_t *gas_used,
                      struct UnmanagedVector *key_out,
                      struct UnmanagedVector *err_msg_out);
  int32_t (*next_value)(struct IteratorReference iterator,
                        struct gas_meter_t *gas_meter,
                        uint64_t *gas_used,
                        struct UnmanagedVector *value_out,
                        struct UnmanagedVector *err_msg_out);
} IteratorVtable;

typedef struct GoIter {
  struct gas_meter_t *gas_meter;
  /**
   * A reference which identifies the iterator and allows finding and accessing the
   * actual iterator instance in Go. Once fully initalized, this is immutable.
   */
  struct IteratorReference reference;
  struct IteratorVtable vtable;
} GoIter;

typedef struct DbVtable {
  int32_t (*read_db)(struct db_t *db,
                     struct gas_meter_t *gas_meter,
                     uint64_t *gas_used,
                     struct U8SliceView key,
                     struct UnmanagedVector *value_out,
                     struct UnmanagedVector *err_msg_out);
  int32_t (*write_db)(struct db_t *db,
                      struct gas_meter_t *gas_meter,
                      uint64_t *gas_used,
                      struct U8SliceView key,
                      struct U8SliceView value,
                      struct UnmanagedVector *err_msg_out);
  int32_t (*remove_db)(struct db_t *db,
                       struct gas_meter_t *gas_meter,
                       uint64_t *gas_used,
                       struct U8SliceView key,
                       struct UnmanagedVector *err_msg_out);
  int32_t (*scan_db)(struct db_t *db,
                     struct gas_meter_t *gas_meter,
                     uint64_t *gas_used,
                     struct U8SliceView start,
                     struct U8SliceView end,
                     int32_t order,
                     struct GoIter *iterator_out,
                     struct UnmanagedVector *err_msg_out);
} DbVtable;

typedef struct Db {
  struct gas_meter_t *gas_meter;
  struct db_t *state;
  struct DbVtable vtable;
} Db;

typedef struct api_t {
  uint8_t _private[0];
} api_t;

typedef struct GoApiVtable {
  int32_t (*humanize_address)(const struct api_t *api,
                              struct U8SliceView input,
                              struct UnmanagedVector *humanized_address_out,
                              struct UnmanagedVector *err_msg_out,
                              uint64_t *gas_used);
  int32_t (*canonicalize_address)(const struct api_t *api,
                                  struct U8SliceView input,
                                  struct UnmanagedVector *canonicalized_address_out,
                                  struct UnmanagedVector *err_msg_out,
                                  uint64_t *gas_used);
  int32_t (*validate_address)(const struct api_t *api,
                              struct U8SliceView input,
                              struct UnmanagedVector *err_msg_out,
                              uint64_t *gas_used);
} GoApiVtable;

typedef struct GoApi {
  const struct api_t *state;
  struct GoApiVtable vtable;
} GoApi;

typedef struct querier_t {
  uint8_t _private[0];
} querier_t;

typedef struct QuerierVtable {
  int32_t (*query_external)(const struct querier_t *querier,
                            uint64_t gas_limit,
                            uint64_t *gas_used,
                            struct U8SliceView request,
                            struct UnmanagedVector *result_out,
                            struct UnmanagedVector *err_msg_out);
} QuerierVtable;

typedef struct GoQuerier {
  const struct querier_t *state;
  struct QuerierVtable vtable;
} GoQuerier;

typedef struct GasReport {
  /**
   * The original limit the instance was created with
   */
  uint64_t limit;
  /**
   * The remaining gas that can be spend
   */
  uint64_t remaining;
  /**
   * The amount of gas that was spend and metered externally in operations triggered by this instance
   */
  uint64_t used_externally;
  /**
   * The amount of gas that was spend and metered internally (i.e. by executing Wasm and calling
   * API methods which are not metered externally)
   */
  uint64_t used_internally;
} GasReport;

struct cache_t *init_cache(struct ByteSliceView config, struct UnmanagedVector *error_msg);

struct UnmanagedVector save_wasm(struct cache_t *cache,
                                 struct ByteSliceView wasm,
                                 bool unchecked,
                                 struct UnmanagedVector *error_msg);

void remove_wasm(struct cache_t *cache,
                 struct ByteSliceView checksum,
                 struct UnmanagedVector *error_msg);

struct UnmanagedVector load_wasm(struct cache_t *cache,
                                 struct ByteSliceView checksum,
                                 struct UnmanagedVector *error_msg);

void pin(struct cache_t *cache, struct ByteSliceView checksum, struct UnmanagedVector *error_msg);

void unpin(struct cache_t *cache, struct ByteSliceView checksum, struct UnmanagedVector *error_msg);

struct AnalysisReport analyze_code(struct cache_t *cache,
                                   struct ByteSliceView checksum,
                                   struct UnmanagedVector *error_msg);

struct Metrics get_metrics(struct cache_t *cache, struct UnmanagedVector *error_msg);

struct UnmanagedVector get_pinned_metrics(struct cache_t *cache, struct UnmanagedVector *error_msg);

/**
 * frees a cache reference
 *
 * # Safety
 *
 * This must be called exactly once for any `*cache_t` returned by `init_cache`
 * and cannot be called on any other pointer.
 */
void release_cache(struct cache_t *cache);

struct UnmanagedVector instantiate(struct cache_t *cache,
                                   struct ByteSliceView checksum,
                                   struct ByteSliceView env,
                                   struct ByteSliceView info,
                                   struct ByteSliceView msg,
                                   struct Db db,
                                   struct GoApi api,
                                   struct GoQuerier querier,
                                   uint64_t gas_limit,
                                   bool print_debug,
                                   struct GasReport *gas_report,
                                   struct UnmanagedVector *error_msg);

struct UnmanagedVector execute(struct cache_t *cache,
                               struct ByteSliceView checksum,
                               struct ByteSliceView env,
                               struct ByteSliceView info,
                               struct ByteSliceView msg,
                               struct Db db,
                               struct GoApi api,
                               struct GoQuerier querier,
                               uint64_t gas_limit,
                               bool print_debug,
                               struct GasReport *gas_report,
                               struct UnmanagedVector *error_msg);

struct UnmanagedVector migrate(struct cache_t *cache,
                               struct ByteSliceView checksum,
                               struct ByteSliceView env,
                               struct ByteSliceView msg,
                               struct Db db,
                               struct GoApi api,
                               struct GoQuerier querier,
                               uint64_t gas_limit,
                               bool print_debug,
                               struct GasReport *gas_report,
                               struct UnmanagedVector *error_msg);

struct UnmanagedVector migrate_with_info(struct cache_t *cache,
                                         struct ByteSliceView checksum,
                                         struct ByteSliceView env,
                                         struct ByteSliceView msg,
                                         struct ByteSliceView migrate_info,
                                         struct Db db,
                                         struct GoApi api,
                                         struct GoQuerier querier,
                                         uint64_t gas_limit,
                                         bool print_debug,
                                         struct GasReport *gas_report,
                                         struct UnmanagedVector *error_msg);

struct UnmanagedVector sudo(struct cache_t *cache,
                            struct ByteSliceView checksum,
                            struct ByteSliceView env,
                            struct ByteSliceView msg,
                            struct Db db,
                            struct GoApi api,
                            struct GoQuerier querier,
                            uint64_t gas_limit,
                            bool print_debug,
                            struct GasReport *gas_report,
                            struct UnmanagedVector *error_msg);

struct UnmanagedVector reply(struct cache_t *cache,
                             struct ByteSliceView checksum,
                             struct ByteSliceView env,
                             struct ByteSliceView msg,
                             struct Db db,
                             struct GoApi api,
                             struct GoQuerier querier,
                             uint64_t gas_limit,
                             bool print_debug,
                             struct GasReport *gas_report,
                             struct UnmanagedVector *error_msg);

struct UnmanagedVector query(struct cache_t *cache,
                             struct ByteSliceView checksum,
                             struct ByteSliceView env,
                             struct ByteSliceView msg,
                             struct Db db,
                             struct GoApi api,
                             struct GoQuerier querier,
                             uint64_t gas_limit,
                             bool print_debug,
                             struct GasReport *gas_report,
                             struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_channel_open(struct cache_t *cache,
                                        struct ByteSliceView checksum,
                                        struct ByteSliceView env,
                                        struct ByteSliceView msg,
                                        struct Db db,
                                        struct GoApi api,
                                        struct GoQuerier querier,
                                        uint64_t gas_limit,
                                        bool print_debug,
                                        struct GasReport *gas_report,
                                        struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_channel_connect(struct cache_t *cache,
                                           struct ByteSliceView checksum,
                                           struct ByteSliceView env,
                                           struct ByteSliceView msg,
                                           struct Db db,
                                           struct GoApi api,
                                           struct GoQuerier querier,
                                           uint64_t gas_limit,
                                           bool print_debug,
                                           struct GasReport *gas_report,
                                           struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_channel_close(struct cache_t *cache,
                                         struct ByteSliceView checksum,
                                         struct ByteSliceView env,
                                         struct ByteSliceView msg,
                                         struct Db db,
                                         struct GoApi api,
                                         struct GoQuerier querier,
                                         uint64_t gas_limit,
                                         bool print_debug,
                                         struct GasReport *gas_report,
                                         struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_packet_receive(struct cache_t *cache,
                                          struct ByteSliceView checksum,
                                          struct ByteSliceView env,
                                          struct ByteSliceView msg,
                                          struct Db db,
                                          struct GoApi api,
                                          struct GoQuerier querier,
                                          uint64_t gas_limit,
                                          bool print_debug,
                                          struct GasReport *gas_report,
                                          struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_packet_ack(struct cache_t *cache,
                                      struct ByteSliceView checksum,
                                      struct ByteSliceView env,
                                      struct ByteSliceView msg,
                                      struct Db db,
                                      struct GoApi api,
                                      struct GoQuerier querier,
                                      uint64_t gas_limit,
                                      bool print_debug,
                                      struct GasReport *gas_report,
                                      struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_packet_timeout(struct cache_t *cache,
                                          struct ByteSliceView checksum,
                                          struct ByteSliceView env,
                                          struct ByteSliceView msg,
                                          struct Db db,
                                          struct GoApi api,
                                          struct GoQuerier querier,
                                          uint64_t gas_limit,
                                          bool print_debug,
                                          struct GasReport *gas_report,
                                          struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_source_callback(struct cache_t *cache,
                                           struct ByteSliceView checksum,
                                           struct ByteSliceView env,
                                           struct ByteSliceView msg,
                                           struct Db db,
                                           struct GoApi api,
                                           struct GoQuerier querier,
                                           uint64_t gas_limit,
                                           bool print_debug,
                                           struct GasReport *gas_report,
                                           struct UnmanagedVector *error_msg);

struct UnmanagedVector ibc_destination_callback(struct cache_t *cache,
                                                struct ByteSliceView checksum,
                                                struct ByteSliceView env,
                                                struct ByteSliceView msg,
                                                struct Db db,
                                                struct GoApi api,
                                                struct GoQuerier querier,
                                                uint64_t gas_limit,
                                                bool print_debug,
                                                struct GasReport *gas_report,
                                                struct UnmanagedVector *error_msg);

struct UnmanagedVector new_unmanaged_vector(bool nil, const uint8_t *ptr, uintptr_t length);

void destroy_unmanaged_vector(struct UnmanagedVector v);

/**
 * Returns a version number of this library as a C string.
 *
 * The string is owned by libwasmvm and must not be mutated or destroyed by the caller.
 */
const char *version_str(void);
