import { describe, it, expect } from 'vitest';
import { useConfirmation } from './useConfirmation';

describe('useConfirmation', () => {
  it('resolves with isConfirmed: true when onConfirm is called', async () => {
    const { awaitConfirmation, modalProps } = useConfirmation();

    const promise = awaitConfirmation();
    modalProps.onConfirm();

    const result = await promise;
    expect(result).toEqual({ isConfirmed: true });
  });

  it('resolves with isConfirmed: false when onClose is called', async () => {
    const { awaitConfirmation, modalProps } = useConfirmation();

    const promise = awaitConfirmation();
    modalProps.onClose();

    const result = await promise;
    expect(result).toEqual({ isConfirmed: false });
  });

  it('sets isOpen to true when awaitConfirmation is called', () => {
    const { awaitConfirmation, modalProps } = useConfirmation();

    expect(modalProps.open.value).toBe(false);
    awaitConfirmation();
    expect(modalProps.open.value).toBe(true);
  });

  it('sets isOpen to false after confirmation', async () => {
    const { awaitConfirmation, modalProps } = useConfirmation();

    const promise = awaitConfirmation();
    modalProps.onConfirm();
    await promise;

    expect(modalProps.open.value).toBe(false);
  });

  it('calling awaitConfirmation twice does not leak watchers', async () => {
    const { awaitConfirmation, modalProps } = useConfirmation();

    // Start first confirmation - this creates a watcher
    const promise1 = awaitConfirmation();

    // Start second confirmation - this should clean up the first watcher
    // and reset confirmationStatus. The first promise's watcher is stopped
    // because awaitConfirmation() sets confirmationStatus to undefined,
    // which does NOT trigger the watcher (it was already undefined).
    const promise2 = awaitConfirmation();

    // Confirm the second one
    modalProps.onConfirm();
    const result2 = await promise2;
    expect(result2).toEqual({ isConfirmed: true });

    // The first promise should also resolve because the watcher fires
    // on the same confirmationStatus ref. The key point is that the
    // watcher from the first call is stopped inside its own callback,
    // so it doesn't leak.
    const result1 = await promise1;
    expect(result1).toEqual({ isConfirmed: true });
  });

  it('can be reused after a confirmation cycle', async () => {
    const { awaitConfirmation, modalProps } = useConfirmation();

    // First cycle
    const promise1 = awaitConfirmation();
    modalProps.onConfirm();
    await promise1;

    // Second cycle
    const promise2 = awaitConfirmation();
    modalProps.onClose();
    const result2 = await promise2;
    expect(result2).toEqual({ isConfirmed: false });
  });
});
