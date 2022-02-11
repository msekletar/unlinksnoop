/* SPDX-License-Identifier: MIT OR GPL-2.0 */
#include "vmlinux.h"

#include <bpf/bpf_helpers.h>
#include <bpf/bpf_tracing.h>

#define MAX_FILENAME_LEN 1024
#define TASK_COMM_LEN 16

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 512 * 1024);
} events SEC(".maps");

struct unlink_event {
	int pid;
	char comm[TASK_COMM_LEN];
	char filename[MAX_FILENAME_LEN];
};

SEC("tracepoint/syscalls/sys_enter_unlinkat")
int handle_unlink(struct trace_event_raw_sys_enter *ctx) {
	struct unlink_event *e;

	e = bpf_ringbuf_reserve(&events, sizeof(*e), 0);
	if (!e) {
		bpf_printk("failed to allocate event struct in ring\n");
		return 0;
	}

	e->pid = bpf_get_current_pid_tgid() >> 32;

	bpf_get_current_comm(&e->comm, sizeof(e->comm));
	bpf_probe_read_str(&e->filename, sizeof(e->filename), (void *)ctx->args[1]);

	bpf_ringbuf_submit(e, 0);

	return 0;
}

static const char _license[] SEC("license") = "Dual MIT/GPL";
