/** One event from the agent, as emitted on the "assistant:event" channel.
 *
 * Wails only generates models for types in a bound method's signature, and
 * these travel over the event bus — so the shape is mirrored here by hand.
 * Keep it in step with assistant.Event in Go.
 */
export interface AgentEvent {
    kind: 'text' | 'tool_call' | 'tool_result' | 'done'
    text?: string
    tool?: string
    input?: string
    result?: string
    failed?: boolean
    mutating?: boolean
}

/** An entry in the visible transcript. */
export type Item =
    | {kind: 'user'; text: string}
    | {kind: 'assistant'; text: string}
    | {kind: 'tool'; tool: string; input: string; result?: string; failed?: boolean; mutating?: boolean}

/** Folds one event into the transcript, returning the new list.
 *
 * Text arrives as a stream of fragments, so it accumulates into the trailing
 * assistant entry rather than creating one per delta. A tool result attaches to
 * the most recent call of that tool still waiting for one.
 */
export function applyEvent(items: Item[], e: AgentEvent): Item[] {
    switch (e.kind) {
        case 'text': {
            if (!e.text) return items
            const last = items[items.length - 1]
            if (last?.kind === 'assistant') {
                return [...items.slice(0, -1), {...last, text: last.text + e.text}]
            }
            return [...items, {kind: 'assistant', text: e.text}]
        }

        case 'tool_call':
            return [...items, {kind: 'tool', tool: e.tool ?? '', input: e.input ?? '', mutating: e.mutating}]

        case 'tool_result': {
            const index = items.findLastIndex(
                i => i.kind === 'tool' && i.tool === e.tool && i.result === undefined,
            )
            if (index < 0) return items
            const target = items[index] as Extract<Item, {kind: 'tool'}>
            const updated: Item = {...target, result: e.result ?? '', failed: e.failed}
            return [...items.slice(0, index), updated, ...items.slice(index + 1)]
        }

        default:
            return items
    }
}

/** Renders a tool call as a one-line summary for the transcript. */
export function describeCall(tool: string, input: string): string {
    try {
        const args = JSON.parse(input || '{}') as Record<string, unknown>
        const parts = Object.entries(args)
            .filter(([k]) => k !== 'content')
            .map(([k, v]) => `${k}=${typeof v === 'string' ? v : JSON.stringify(v)}`)
        // write_file carries the whole file in `content`; show its size instead.
        if (typeof args.content === 'string') parts.push(`${args.content.length} bytes`)
        return `${tool}(${parts.join(', ')})`
    } catch {
        return `${tool}(${input})`
    }
}
