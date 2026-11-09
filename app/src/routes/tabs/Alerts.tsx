import { useState } from 'react';

import { errorMessage, type AlertRule } from '@/api/client';
import { AlertRuleForm } from '@/components/AlertRuleForm';
import { Button, Card, Chip, ErrorText } from '@/components/ui';
import { useAlertEvents, useAlertMutations, useAlertRules, useChannels } from '@/hooks/useAlerts';
import { useCanEdit } from '@/hooks/useDashboards';
import { useDocumentTitle } from '@/hooks/useDocumentTitle';
import { describeCondition, formatValue, stateColor, type AlertState } from '@/lib/alerts';
import { mono, useTheme } from '@/theme';

function StateBadge({ state }: { state: AlertState | 'resolved' }) {
  const t = useTheme();
  const c = stateColor(state, t);
  return (
    <span
      aria-label={`State ${state}`}
      style={{ borderColor: c, borderWidth: 1, borderStyle: 'solid', borderRadius: 10, padding: '2px 8px', color: c, fontSize: 12, fontWeight: 700, textTransform: 'uppercase' }}>
      {state}
    </span>
  );
}

function RuleCard({ rule, canEdit, onEdit }: { rule: AlertRule; canEdit: boolean; onEdit: () => void }) {
  const t = useTheme();
  const { deleteRule, updateRule } = useAlertMutations();
  const [error, setError] = useState<string | null>(null);
  const toggle = () =>
    updateRule.mutate(
      {
        id: rule.id,
        body: {
          name: rule.name, query: rule.query, comparator: rule.comparator, threshold: rule.threshold, window: rule.window,
          for_duration: rule.for_duration, cooldown: rule.cooldown, severity: rule.severity, enabled: !rule.enabled,
          notify_channel_ids: rule.notify_channel_ids,
        },
      },
      { onError: (e) => setError(errorMessage(e, 'Could not update the rule')) },
    );
  return (
    <Card style={{ display: 'flex', flexDirection: 'column', gap: 6, opacity: rule.enabled ? 1 : 0.6 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
        <span style={{ color: t.text, fontSize: 16, fontWeight: 600, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
          {rule.name}
        </span>
        <StateBadge state={rule.enabled ? rule.state.state : 'ok'} />
      </div>
      <span style={{ ...mono, color: t.textDim, fontSize: 12, overflow: 'hidden', display: '-webkit-box', WebkitLineClamp: 2, WebkitBoxOrient: 'vertical' }}>
        {rule.query}
      </span>
      <span style={{ color: t.textDim, fontSize: 12 }}>
        {describeCondition(rule)} · {rule.severity}
        {rule.enabled ? '' : ' · disabled'}
      </span>
      {rule.enabled && rule.state.last_eval && (
        <span style={{ color: t.textDim, fontSize: 12 }}>Last value {formatValue(rule.state.last_value)}</span>
      )}
      {rule.state.last_error ? <ErrorText>{rule.state.last_error}</ErrorText> : null}
      {error && <ErrorText>{error}</ErrorText>}
      {canEdit && (
        <div style={{ display: 'flex', gap: 8, flexWrap: 'wrap' }}>
          <Chip label="Edit" accessibilityLabel={`Edit ${rule.name}`} onPress={onEdit} />
          <Chip label={rule.enabled ? 'Disable' : 'Enable'} accessibilityLabel={`${rule.enabled ? 'Disable' : 'Enable'} ${rule.name}`} onPress={toggle} />
          <Chip
            label="Delete"
            accessibilityLabel={`Delete ${rule.name}`}
            onPress={() => deleteRule.mutate(rule.id, { onError: (e) => setError(errorMessage(e, 'Could not delete the rule')) })}
          />
        </div>
      )}
    </Card>
  );
}

export function Alerts() {
  useDocumentTitle('Alerts');
  const t = useTheme();
  const canEdit = useCanEdit();
  const rules = useAlertRules();
  const events = useAlertEvents();
  const channels = useChannels(canEdit);
  const { createRule, updateRule } = useAlertMutations();
  const [editing, setEditing] = useState<AlertRule | 'new' | null>(null);

  const chans = channels.data ?? [];
  return (
    <div style={{ padding: 16, display: 'flex', flexDirection: 'column', gap: 12 }}>
      {canEdit && editing === null && <Button title="New alert rule" onPress={() => setEditing('new')} />}
      {editing === 'new' && (
        <AlertRuleForm channels={chans} submitLabel="Create rule" onCancel={() => setEditing(null)} onSubmit={async (body) => { await createRule.mutateAsync(body); setEditing(null); }} />
      )}
      {rules.isError && <ErrorText>{errorMessage(rules.error, 'Could not load alert rules')}</ErrorText>}
      {rules.data?.length === 0 && editing === null && <span style={{ color: t.textDim }}>No alert rules yet.</span>}
      {rules.data?.map((r) =>
        typeof editing === 'object' && editing?.id === r.id ? (
          <AlertRuleForm
            key={r.id}
            initial={r}
            channels={chans}
            submitLabel="Save rule"
            onCancel={() => setEditing(null)}
            onSubmit={async (body) => { await updateRule.mutateAsync({ id: r.id, body }); setEditing(null); }}
          />
        ) : (
          <RuleCard key={r.id} rule={r} canEdit={canEdit} onEdit={() => setEditing(r)} />
        ),
      )}

      <span style={{ color: t.text, fontSize: 16, fontWeight: 700, marginTop: 8 }}>History</span>
      {events.data?.length === 0 && <span style={{ color: t.textDim }}>No alert activity yet.</span>}
      {events.data?.map((e) => (
        <Card key={e.id} style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
            <StateBadge state={e.state} />
            <span style={{ color: t.text, flex: 1, overflow: 'hidden', textOverflow: 'ellipsis', whiteSpace: 'nowrap' }}>
              {e.rule_name || 'Deleted rule'}
            </span>
          </div>
          <span style={{ color: t.textDim, fontSize: 12 }}>
            {new Date(e.at).toLocaleString()} · value {formatValue(e.value)}
            {e.notified ? ' · notified' : ''}
          </span>
          {e.notify_error ? <ErrorText>{e.notify_error}</ErrorText> : null}
        </Card>
      ))}
    </div>
  );
}
