import React, { useEffect, useState } from 'react';
import { Modal } from '@douyinfe/semi-ui';
import { API, showError } from '../../helpers';
import InviteOverviewDetails from './InviteOverviewDetails';

const AdminInviteOverviewModal = ({ t, visible, userId, onCancel, title }) => {
  const [loading, setLoading] = useState(false);
  const [overview, setOverview] = useState(null);

  useEffect(() => {
    if (!visible || !userId) {
      return;
    }
    let mounted = true;
    const loadOverview = async () => {
      setLoading(true);
      try {
        const res = await API.get(`/api/user/${userId}/invite-overview`);
        const { success, message, data } = res.data;
        if (!mounted) return;
        if (success) {
          setOverview(data || null);
        } else {
          showError(message || t('加载邀请明细失败'));
        }
      } catch (error) {
        if (mounted) {
          showError(t('加载邀请明细失败'));
        }
      } finally {
        if (mounted) {
          setLoading(false);
        }
      }
    };
    loadOverview();
    return () => {
      mounted = false;
    };
  }, [visible, userId, t]);

  return (
    <Modal
      title={title || t('邀请明细')}
      visible={visible}
      onCancel={onCancel}
      footer={null}
      size='full-width'
    >
      <InviteOverviewDetails t={t} loading={loading} overview={overview} />
    </Modal>
  );
};

export default AdminInviteOverviewModal;
