# Go Test Catalog

**Total Tests:** 113

**Numbered Tests:** 113

**Unnumbered Tests:** 0

**Numbered Tests Missing Descriptions:** 0

**Numbering Mismatches:** 0

All numbered test numbers are unique.

This catalog lists all tests in the Go codebase.

| Test # | Function Name | Description | File |
|--------|---------------|-------------|------|
| test001 | `Test001_op_execution` | TEST001: Run Op::perform and verify the returned value matches what the op was configured with | op_test.go:26 |
| test002 | `Test002_op_with_contexts` | TEST002: Verify Op reads from DryContext and produces a formatted result using that data | op_test.go:55 |
| test003 | `Test003_op_default_rollback` | TEST003: Confirm that the default rollback implementation is a no-op that always succeeds | op_test.go:79 |
| test004 | `Test004_op_custom_rollback` | TEST004: Verify a custom rollback implementation is called and sets the rolled_back flag | op_test.go:109 |
| test005 | `Test005_perform_with_auto_logging` | TEST005: Confirm the Perform() utility wraps an op with automatic logging and returns its result | ops_test.go:11 |
| test006 | `Test006_caller_trigger_name` | TEST006: Verify GetCallerTriggerName returns a string containing the file name with a "." separator | ops_test.go:26 |
| test007 | `Test007_wrap_nested_op_exception` | TEST007: Confirm WrapNestedOpException wraps an error with the op name in the message | ops_test.go:37 |
| test008 | `Test008_wrap_runtime_exception` | TEST008: Verify WrapRuntimeException converts any error into an OpError::ExecutionFailed | ops_test.go:57 |
| test009 | `Test009_dry_context_basic_operations` | TEST009: Insert typed values into DryContext and verify get/contains work correctly | context_test.go:14 |
| test010 | `Test010_dry_context_builder` | TEST010: Build a DryContext with chained WithValue calls and verify all values are stored | context_test.go:34 |
| test011 | `Test011_wet_context_basic_operations` | TEST011: Insert a reference into WetContext and retrieve it by type via WetGetRef | context_test.go:48 |
| test012 | `Test012_wet_context_builder` | TEST012: Build a WetContext with chained WithRef calls and verify contains for each key | context_test.go:65 |
| test013 | `Test013_required_values` | TEST013: Confirm get_required succeeds for present keys and returns an error for missing keys | context_test.go:77 |
| test014 | `Test014_context_merge` | TEST014: Merge two DryContexts and verify values from both are accessible in the target | context_test.go:91 |
| test015 | `Test015_dry_context_type_mismatch_error` | TEST015: Verify get_required returns a Type mismatch error when the stored type doesn't match | context_test.go:105 |
| test016 | `Test016_wet_context_type_mismatch_error` | TEST016: Verify WetContext get_required returns a Type mismatch error when the stored ref type differs | context_test.go:151 |
| test017 | `Test017_control_flags` | TEST017: Set and clear abort flags on DryContext and verify IsAborted and AbortReason reflect state | context_test.go:180 |
| test018 | `Test018_control_flags_merge` | TEST018: Merge contexts with abort flags and confirm the target inherits the abort state correctly | context_test.go:209 |
| test019 | `Test019_get_or_insert_with` | TEST019: Verify DryGetOrInsertWith inserts when missing and returns existing without calling factory | context_test.go:240 |
| test020 | `Test020_get_or_compute_with` | TEST020: Verify DryGetOrComputeWith computes and stores a value using context data | context_test.go:270 |
| test021 | `Test021_metadata_builder` | TEST021: Build OpMetadata with name, description, and schemas and verify all fields are populated | metadata_test.go:11 |
| test022 | `Test022_trigger_fuse` | TEST022: Construct a TriggerFuse with data and verify the trigger name and dry context values | metadata_test.go:42 |
| test023 | `Test023_basic_validation` | TEST023: Validate a DryContext against an input schema and confirm valid/invalid reports | metadata_test.go:63 |
| test024 | `Test024_simple_flat_outline` | TEST024: Build a flat ListingOutline with depth-0 entries and verify max_depth, levels, and flatten count | structured_queries_test.go:11 |
| test025 | `Test025_hierarchical_outline` | TEST025: Build a two-level outline with chapters and sections and verify depth, level counts, and flatten | structured_queries_test.go:34 |
| test026 | `Test026_complex_part_based_outline` | TEST026: Build a three-level part/chapter/section outline and verify depth and per-level entry counts | structured_queries_test.go:62 |
| test027 | `Test027_flatten_preserves_hierarchy` | TEST027: Flatten a nested outline and verify each entry's path reflects its ancestry correctly | structured_queries_test.go:99 |
| test028 | `Test028_schema_generation` | TEST028: Call GenerateOutlineSchema and verify the returned JSON contains all required definitions | structured_queries_test.go:127 |
| test029 | `Test029_logging_wrapper_success` | TEST029: Wrap a successful op in LoggingWrapper and verify it passes through the result unchanged | logging_wrapper_test.go:41 |
| test030 | `Test030_logging_wrapper_failure` | TEST030: Wrap a failing op in LoggingWrapper and verify the error includes the op name context | logging_wrapper_test.go:56 |
| test031 | `Test031_context_aware_logger` | TEST031: Use CreateContextAwareLogger helper and verify the wrapped op returns its result | logging_wrapper_test.go:78 |
| test032 | `Test032_ansi_color_constants` | TEST032: Verify ANSI color escape code constants have the expected ANSI sequence values | logging_wrapper_test.go:93 |
| test033 | `Test033_timeout_wrapper_success` | TEST033: Wrap a fast op in TimeBoundWrapper and confirm it completes before the timeout | timeout_wrapper_test.go:64 |
| test034 | `Test034_timeout_wrapper_timeout` | TEST034: Wrap a slow op in TimeBoundWrapper with a short timeout and verify a Timeout error is returned | timeout_wrapper_test.go:79 |
| test035 | `Test035_timeout_wrapper_with_name` | TEST035: Create a named TimeBoundWrapper and verify the op succeeds and returns the expected value | timeout_wrapper_test.go:101 |
| test036 | `Test036_caller_name_wrapper` | TEST036: Use CreateTimeoutWrapperWithCallerName helper and verify the op result is returned | timeout_wrapper_test.go:116 |
| test037 | `Test037_logged_timeout_wrapper` | TEST037: Use CreateLoggedTimeoutWrapper to compose logging and timeout wrappers and verify success | timeout_wrapper_test.go:131 |
| test038 | `Test038_valid_input_output` | TEST038: Run ValidatingWrapper with a valid input and verify the op executes and returns the result | validating_wrapper_test.go:48 |
| test039 | `Test039_invalid_input_missing_required` | TEST039: Run ValidatingWrapper without a required input field and verify a Context validation error | validating_wrapper_test.go:64 |
| test040 | `Test040_invalid_input_out_of_range` | TEST040: Run ValidatingWrapper with an input exceeding the schema maximum and verify a validation error | validating_wrapper_test.go:84 |
| test041 | `Test041_input_only_validation` | TEST041: Use NewValidatingWrapperInputOnly and confirm input is validated while output is not | validating_wrapper_test.go:105 |
| test042 | `Test042_output_only_validation` | TEST042: Use NewValidatingWrapperOutputOnly and confirm output is validated while input is not | validating_wrapper_test.go:139 |
| test043 | `Test043_no_schema_validation` | TEST043: Wrap an op with no schemas in ValidatingWrapper and confirm it still succeeds | validating_wrapper_test.go:174 |
| test044 | `Test044_metadata_transparency` | TEST044: Verify ValidatingWrapper::Metadata() delegates to the inner op's metadata unchanged | validating_wrapper_test.go:201 |
| test045 | `Test045_reference_validation` | TEST045: Verify ValidatingWrapper checks reference_schema and rejects when required refs are missing | validating_wrapper_test.go:220 |
| test046 | `Test046_no_reference_schema` | TEST046: Wrap an op with no reference schema in ValidatingWrapper and confirm it succeeds | validating_wrapper_test.go:289 |
| test047 | `Test047_batch_metadata_with_data_flow` | TEST047: Build BatchMetadata from producer/consumer ops and verify only external inputs are required | batch_metadata_test.go:69 |
| test048 | `Test048_reference_schema_merging` | TEST048: Build BatchMetadata from two ops with different reference schemas and verify union of required refs | batch_metadata_test.go:140 |
| test049 | `Test049_batch_op_success` | TEST049: Run BatchOp with two succeeding ops and verify results contain both values in order | batch_test.go:30 |
| test050 | `Test050_batch_op_failure` | TEST050: Run BatchOp where the second op fails and verify the batch returns an error | batch_test.go:49 |
| test051 | `Test051_batch_op_returns_all_results` | TEST051: Run BatchOp with two ops and verify both result values are present in order | batch_test.go:65 |
| test052 | `Test052_batch_metadata_data_flow` | TEST052: Verify BatchOp metadata correctly identifies only the externally-required input fields | batch_test.go:96 |
| test053 | `Test053_batch_reference_schema_merging` | TEST053: Verify BatchOp merges reference schemas from all ops into a unified set of required refs | batch_test.go:142 |
| test054 | `Test054_batch_rollback_on_failure` | TEST054: Run BatchOp where the third op fails and verify rollback is called on first two but not the third | batch_test.go:221 |
| test055 | `Test055_batch_rollback_order` | TEST055: Run BatchOp where the last op fails and verify rollback occurs in reverse (LIFO) order | batch_test.go:294 |
| test056 | `Test056_batch_rollback_on_failure_partial` | TEST056: Run BatchOp where one op fails and verify rollback is triggered for succeeded ops | batch_test.go:364 |
| test057 | `Test057_abort_without_reason` | TEST057: Invoke Abort without a reason and verify the context is aborted with no reason string | control_flow_test.go:64 |
| test058 | `Test058_abort_with_reason` | TEST058: Invoke AbortWithReason and verify abort_reason matches | control_flow_test.go:90 |
| test059 | `Test059_continue_loop` | TEST059: Use ContinueLoop inside an op and verify the scoped continue flag is set in context | control_flow_test.go:117 |
| test060 | `Test060_check_abort` | TEST060: Use CheckAbort to short-circuit when the abort flag is already set in context | control_flow_test.go:144 |
| test061 | `Test061_batch_op_with_abort` | TEST061: Run a BatchOp where the second op aborts and verify the batch stops and propagates the abort | control_flow_test.go:176 |
| test062 | `Test062_batch_op_with_pre_existing_abort` | TEST062: Start a BatchOp with an abort flag already set and verify it immediately returns Aborted | control_flow_test.go:201 |
| test063 | `Test063_loop_op_with_continue` | TEST063: Run a LoopOp where an op signals continue and verify subsequent ops in the iteration are skipped | control_flow_test.go:227 |
| test064 | `Test064_loop_op_with_abort` | TEST064: Run a LoopOp where an op aborts mid-loop and verify the loop terminates with the abort error | control_flow_test.go:255 |
| test065 | `Test065_loop_op_with_pre_existing_abort` | TEST065: Start a LoopOp with an abort flag already set and verify it immediately returns Aborted | control_flow_test.go:280 |
| test066 | `Test066_complex_control_flow_scenario` | TEST066: Nest a batch with a continue op inside a loop and verify results across all iterations | control_flow_test.go:303 |
| test067 | `Test067_loop_op_basic` | TEST067: Run a LoopOp for 3 iterations with 2 ops each and verify all 6 results in order | loop_op_test.go:37 |
| test068 | `Test068_loop_op_with_counter_access` | TEST068: Run a LoopOp where each op reads the loop counter and verify values are 0, 1, 2 | loop_op_test.go:62 |
| test069 | `Test069_loop_op_existing_counter` | TEST069: Start a LoopOp with a pre-initialized counter and verify it only executes the remaining iterations | loop_op_test.go:84 |
| test070 | `Test070_loop_op_zero_limit` | TEST070: Run a LoopOp with a zero iteration limit and verify no ops are executed | loop_op_test.go:104 |
| test071 | `Test071_loop_op_builder_pattern` | TEST071: Build a LoopOp with AddOp chaining and verify all added ops run across all iterations | loop_op_test.go:120 |
| test072 | `Test072_loop_op_rollback_on_iteration_failure` | TEST072: Run a LoopOp where the third op fails and verify succeeded ops are rolled back in reverse order | loop_op_test.go:144 |
| test073 | `Test073_loop_op_rollback_order_within_iteration` | TEST073: Run a LoopOp where the last op fails and verify rollback occurs in LIFO order within the iteration | loop_op_test.go:209 |
| test074 | `Test074_loop_op_successful_iterations_not_rolled_back` | TEST074: Run a LoopOp that fails on iteration 2 and verify previously completed iterations are not rolled back | loop_op_test.go:269 |
| test075 | `Test075_loop_op_mixed_iteration_with_rollback` | TEST075: Run a LoopOp where op2 fails on iteration 1 and verify only op1 from that iteration is rolled back | loop_op_test.go:336 |
| test076 | `Test076_loop_op_continue_on_error` | TEST076: Run a LoopOp configured to continue on error and verify subsequent iterations still execute | loop_op_test.go:414 |
| test077 | `Test077_dry_put_and_get` | TEST077: Use DryPut and DryGet helper functions to store and retrieve a typed value by variable name | macro_test.go:16 |
| test078 | `Test078_dry_require` | TEST078: Use DryGetRequired to retrieve a required value and verify error when key is missing | macro_test.go:32 |
| test079 | `Test079_dry_result` | TEST079: Use DryResult to store a final result and verify it is stored under both "result" and op name | macro_test.go:54 |
| test080 | `Test080_wet_put_ref_and_require_ref` | TEST080: Use WetPutRef and WetGetRefRequired to store and retrieve a service reference | macro_test.go:72 |
| test081 | `Test081_wet_put_arc` | TEST081: Use WetPutArc to store a shared service reference and retrieve it via WetGetRefRequired | macro_test.go:88 |
| test082 | `Test082_helpers_in_op` | TEST082: Run a full op that uses DryGetRequired and WetGetRefRequired internally and verify the output | macro_test.go:129 |
| test083 | `Test083_error_handling_and_wrapper_chains` | TEST083: Compose timeout and logging wrappers around a failing op and verify the error message includes the op name | integration_test.go:37 |
| test084 | `Test084_stack_trace_analysis` | TEST084: Call GetCallerTriggerName from within a test and verify it reflects the integration test module path | integration_test.go:59 |
| test085 | `Test085_exception_wrapping_utilities` | TEST085: Use WrapNestedOpException and verify the wrapped error contains both op name and original message | integration_test.go:70 |
| test086 | `Test086_timeout_wrapper_functionality` | TEST086: Wrap a slow op in a short-timeout TimeBoundWrapper and verify the error is wrapped with logging context | integration_test.go:87 |
| test087 | `Test087_dry_and_wet_context_usage` | TEST087: Run an op that retrieves a service from WetContext and reads config values from it | integration_test.go:141 |
| test088 | `Test088_batch_ops` | TEST088: Run a BatchOp with two identical user-building ops and verify both produce the expected User struct | integration_test.go:196 |
| test089 | `Test089_wrapper_composition` | TEST089: Compose TimeBoundWrapper and LoggingWrapper around a simple op and verify the result passes through | integration_test.go:239 |
| test090 | `Test090_perform_utility` | TEST090: Use the Perform() utility function directly and verify it returns the op result with auto-logging | integration_test.go:266 |
| test093 | `Test093_batch_len_and_is_empty` | TEST093: Call BatchOp::Len and IsEmpty on empty and non-empty batches | batch_test.go:396 |
| test094 | `Test094_batch_add_op` | TEST094: Use AddOp to dynamically add an op and verify it is executed | batch_test.go:415 |
| test095 | `Test095_batch_continue_on_error` | TEST095: Run BatchOp::with_continue_on_error and verify it collects results past failures | batch_test.go:431 |
| test096 | `Test096_empty_batch_returns_empty` | TEST096: Run an empty BatchOp and verify it returns an empty result vec | batch_test.go:452 |
| test097 | `Test097_nested_batch_rollback` | TEST097: Verify nested BatchOp rollback propagates correctly when outer batch fails | batch_test.go:467 |
| test098 | `Test098_dry_context_merge_overwrites_keys` | TEST098: Merge two DryContexts where keys overlap and verify the merging context's values win | context_test.go:308 |
| test099 | `Test099_wet_context_merge` | TEST099: Merge two WetContexts and verify both sets of references are accessible in the target | context_test.go:328 |
| test100 | `Test100_dry_context_serde_roundtrip` | TEST100: Serialize and deserialize a DryContext and verify all values survive the round-trip | context_test.go:345 |
| test101 | `Test101_dry_context_clone_is_independent` | TEST101: Clone a DryContext and verify the clone is independent (mutations don't propagate) | context_test.go:373 |
| test102 | `Test102_dry_context_keys` | TEST102: Verify DryContext::keys() returns all inserted keys | context_test.go:387 |
| test103 | `Test103_wet_context_keys` | TEST103: Verify WetContext::keys() returns all inserted reference keys | context_test.go:401 |
| test104 | `Test104_op_error_display_execution_failed` | TEST104: Verify OpError::ExecutionFailed displays with the correct message format | error_test.go:12 |
| test105 | `Test105_op_error_display_timeout` | TEST105: Verify OpError::Timeout displays with the correct timeout_ms value | error_test.go:20 |
| test106 | `Test106_op_error_display_context` | TEST106: Verify OpError::Context displays with the correct message format | error_test.go:28 |
| test107 | `Test107_op_error_display_aborted` | TEST107: Verify OpError::Aborted displays with the correct message format | error_test.go:36 |
| test108 | `Test108_op_error_copy_execution_failed` | TEST108: Copy an OpError::ExecutionFailed and verify the copy is identical | error_test.go:44 |
| test109 | `Test109_op_error_copy_timeout` | TEST109: Copy OpError::Timeout and verify timeout_ms is preserved | error_test.go:59 |
| test110 | `Test110_op_error_copy_other_preserves_message` | TEST110: Copy an OpError wrapping a std error and verify the error message is accessible | error_test.go:71 |
| test111 | `Test111_op_error_from_json_error` | TEST111: Convert a json.Unmarshal error into OpError via conversion function | error_test.go:81 |
| test112 | `Test112_output_only_still_validates_references` | TEST112: Verify ValidatingWrapper::output_only validates references even when input validation is disabled | validating_wrapper_test.go:316 |
| test113 | `Test113_loop_op_break_terminates_loop` | TEST113: Run a LoopOp where an op sets the break flag and verify the loop terminates early | loop_op_test.go:522 |
| test114 | `Test114_loop_op_continue_on_error_skips_failed_iterations` | TEST114: Run LoopOp with_continue_on_error where an op fails and verify the loop continues | loop_op_test.go:543 |
| test115 | `Test115_loop_op_with_no_ops_produces_no_results` | TEST115: Run an empty LoopOp with a non-zero limit and verify it produces no results | loop_op_test.go:604 |
---

*Generated from Go source tree*
*Total tests: 113*
*Total numbered tests: 113*
*Total unnumbered tests: 0*
*Total numbered tests missing descriptions: 0*
*Total numbering mismatches: 0*
