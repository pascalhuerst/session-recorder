import { ref, watch } from 'vue';

export const useConfirmation = () => {
  const isOpen = ref(false);
  const confirmationStatus = ref<'confirmed' | 'rejected'>();

  return {
    awaitConfirmation: async (): Promise<{ isConfirmed: boolean }> => {
      isOpen.value = true;
      confirmationStatus.value = undefined;

      return new Promise((resolve) => {
        const stop = watch(confirmationStatus, () => {
          stop();
          resolve({ isConfirmed: confirmationStatus.value === 'confirmed' });
        });
      });
    },
    modalProps: {
      open: isOpen,
      onConfirm: () => {
        confirmationStatus.value = 'confirmed';
        isOpen.value = false;
      },
      onClose: () => {
        confirmationStatus.value = 'rejected';
        isOpen.value = false;
      },
    },
  };
};
