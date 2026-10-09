<script lang="ts">
  // ChatSecurityBanners — S/MIME and PGP status plus the read-receipt prompt
  // for one message, with the same states and strings as the classic viewer.
  import Icon from '@iconify/svelte'
  import { _ } from '$lib/i18n'
  import type { message as messageModels } from '../../../../wailsjs/go/models'
  import type { ChatThread } from './chatThread.svelte'

  interface Props {
    msg: messageModels.Message
    thread: ChatThread
  }

  let { msg, thread }: Props = $props()

  type Tone = 'info' | 'ok' | 'warn' | 'bad' | 'muted'
  interface Banner {
    tone: Tone
    icon: string
    text: string
    spin?: boolean
  }

  const TONES: Record<Tone, string> = {
    info: 'bg-blue-50 dark:bg-blue-950/30 border-blue-200 dark:border-blue-800 text-blue-700 dark:text-blue-300',
    ok: 'bg-green-50 dark:bg-green-950/30 border-green-200 dark:border-green-800 text-green-700 dark:text-green-300',
    warn: 'bg-amber-50 dark:bg-amber-950/30 border-amber-200 dark:border-amber-800 text-amber-700 dark:text-amber-300',
    bad: 'bg-red-50 dark:bg-red-950/30 border-red-200 dark:border-red-800 text-red-700 dark:text-red-300',
    muted: 'bg-muted/50 border-border text-muted-foreground',
  }

  // status → [tone, icon, i18n key]; keys taking {email} or {keyId} get them filled in.
  const SMIME: Record<string, [Tone, string, string]> = {
    signed: ['ok', 'mdi:shield-check', 'viewer.smimeSignedBy'],
    unknown_signer: ['warn', 'mdi:shield-alert', 'viewer.smimeUnknownSigner'],
    self_signed: ['warn', 'mdi:shield-alert', 'viewer.smimeSelfSigned'],
    expired_cert: ['bad', 'mdi:shield-off', 'viewer.smimeExpiredCert'],
    signer_mismatch: ['warn', 'mdi:shield-alert', 'viewer.smimeSignerMismatch'],
    invalid: ['bad', 'mdi:shield-off', 'viewer.smimeInvalid'],
    decrypt_failed: ['bad', 'mdi:lock-off', 'viewer.smimeDecryptFailed'],
  }
  const PGP: Record<string, [Tone, string, string]> = {
    signed: ['ok', 'mdi:key-check', 'viewer.pgpSignedBy'],
    unknown_key: ['warn', 'mdi:key-alert', 'viewer.pgpUnknownKey'],
    expired_key: ['warn', 'mdi:key-alert', 'viewer.pgpExpiredKey'],
    revoked_key: ['bad', 'mdi:key-remove', 'viewer.pgpRevokedKey'],
    signer_mismatch: ['warn', 'mdi:key-alert', 'viewer.pgpSignerMismatch'],
    invalid: ['bad', 'mdi:key-remove', 'viewer.pgpInvalid'],
    decrypt_failed: ['bad', 'mdi:lock-off', 'viewer.pgpDecryptFailed'],
  }

  const banners = $derived.by(() => {
    const out: Banner[] = []
    const unknown = $_('viewer.unknown').toLowerCase()

    // S/MIME: on-view result for S/MIME messages, cached status otherwise.
    const sm = thread.smimeResults[msg.id]
    if (thread.smimeLoading.has(msg.id)) out.push({ tone: 'muted', icon: 'mdi:loading', text: $_('viewer.processingSMIME'), spin: true })
    if (sm?.smimeEncrypted) out.push({ tone: 'info', icon: 'mdi:lock-check', text: $_('viewer.smimeEncryptedWith') })
    const smStatus = msg.hasSMIME ? sm?.smimeStatus : msg.smimeStatus
    const smEntry = smStatus ? SMIME[smStatus] : undefined
    if (smEntry) {
      const email = (msg.hasSMIME ? sm?.smimeSignerEmail || sm?.smimeSignerSubject : msg.smimeSignerEmail || msg.smimeSignerSubject) || unknown
      out.push({ tone: smEntry[0], icon: smEntry[1], text: $_(smEntry[2], { values: { email } }) })
    }

    const pg = thread.pgpResults[msg.id]
    if (thread.pgpLoading.has(msg.id)) out.push({ tone: 'muted', icon: 'mdi:loading', text: $_('viewer.processingPGP'), spin: true })
    if (pg?.pgpEncrypted) out.push({ tone: 'info', icon: 'mdi:lock-check', text: $_('viewer.pgpEncryptedWith') })
    const pgEntry = pg?.pgpStatus ? PGP[pg.pgpStatus] : undefined
    if (pgEntry) {
      const values = { email: pg!.pgpSignerEmail || unknown, keyId: pg!.pgpSignerKeyId || '' }
      out.push({ tone: pgEntry[0], icon: pgEntry[1], text: $_(pgEntry[2], { values }) })
    }
    return out
  })

  const showReceipt = $derived(thread.shouldShowReadReceipt(msg))
  const sending = $derived(thread.sendingReadReceipt.has(msg.id))
</script>

{#if showReceipt && thread.readReceiptPolicy === 'ask'}
  <div class="flex flex-wrap items-center justify-between gap-2 px-3 py-2 border rounded-md text-sm {TONES.info}">
    <span class="flex items-center gap-2">
      <Icon icon="mdi:email-check-outline" class="w-4 h-4 shrink-0" />
      {$_('viewer.readReceiptRequested')}
    </span>
    <span class="flex items-center gap-2">
      <button
        class="px-3 py-1 text-xs font-medium text-white bg-blue-600 hover:bg-blue-700 rounded transition-colors disabled:opacity-50"
        disabled={sending}
        onclick={() => thread.sendReadReceipt(msg)}
      >
        {#if sending}<Icon icon="mdi:loading" class="w-3 h-3 animate-spin" />{:else}{$_('viewer.sendReceipt')}{/if}
      </button>
      <button class="px-3 py-1 text-xs font-medium rounded hover:bg-blue-100 dark:hover:bg-blue-900/50 transition-colors" onclick={() => thread.ignoreReadReceipt(msg)}>
        {$_('viewer.ignoreReceipt')}
      </button>
    </span>
  </div>
{:else if showReceipt && thread.readReceiptPolicy === 'always' && sending}
  <div class="flex items-center gap-2 px-3 py-2 border rounded-md text-sm {TONES.ok}">
    <Icon icon="mdi:loading" class="w-4 h-4 animate-spin" />
    {$_('viewer.sendingReadReceipt')}
  </div>
{/if}

{#each banners as b (b.text)}
  <div class="flex items-center gap-2 px-3 py-1.5 border rounded-md text-xs {TONES[b.tone]}">
    <Icon icon={b.icon} class="w-4 h-4 shrink-0 {b.spin ? 'animate-spin' : ''}" />
    <span>{b.text}</span>
  </div>
{/each}
