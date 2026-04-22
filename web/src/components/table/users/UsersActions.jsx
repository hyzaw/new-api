/*
Copyright (C) 2025 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/

import React from 'react';
import { Button, Modal, Select, Space, Tag, Typography } from '@douyinfe/semi-ui';

const { Text } = Typography;

const UsersActions = ({
  setShowAddUser,
  selectedUsers,
  setSelectedUsers,
  groupOptions,
  batchSwitchUserGroup,
  migrateUserGroup,
  t,
}) => {
  const [showBatchGroupModal, setShowBatchGroupModal] = React.useState(false);
  const [targetGroup, setTargetGroup] = React.useState('');
  const [showMigrateGroupModal, setShowMigrateGroupModal] = React.useState(false);
  const [sourceGroup, setSourceGroup] = React.useState('');
  const [migrateTargetGroup, setMigrateTargetGroup] = React.useState('');

  // Add new user
  const handleAddUser = () => {
    setShowAddUser(true);
  };

  const handleOpenBatchGroupModal = () => {
    if (!selectedUsers.length) {
      return;
    }
    setTargetGroup('');
    setShowBatchGroupModal(true);
  };

  const handleConfirmBatchGroup = async () => {
    const success = await batchSwitchUserGroup(targetGroup);
    if (!success) {
      return;
    }
    setShowBatchGroupModal(false);
    setTargetGroup('');
  };

  const handleConfirmMigrateGroup = async () => {
    const success = await migrateUserGroup(sourceGroup, migrateTargetGroup);
    if (!success) {
      return;
    }
    setShowMigrateGroupModal(false);
    setSourceGroup('');
    setMigrateTargetGroup('');
  };

  return (
    <>
      <div className='flex flex-wrap gap-2 w-full md:w-auto order-2 md:order-1'>
        <Button className='w-full md:w-auto' onClick={handleAddUser} size='small'>
          {t('添加用户')}
        </Button>

        <Button
          className='w-full md:w-auto'
          onClick={handleOpenBatchGroupModal}
          size='small'
          disabled={!selectedUsers.length}
        >
          {t('批量切换分组')}
          {selectedUsers.length > 0 ? ` (${selectedUsers.length})` : ''}
        </Button>

        <Button
          className='w-full md:w-auto'
          onClick={() => {
            setSourceGroup('');
            setMigrateTargetGroup('');
            setShowMigrateGroupModal(true);
          }}
          size='small'
        >
          {t('按分组整批切换')}
        </Button>

        {selectedUsers.length > 0 && (
          <Button
            theme='borderless'
            className='w-full md:w-auto'
            onClick={() => setSelectedUsers([])}
            size='small'
          >
            {t('清空选择')}
          </Button>
        )}
      </div>

      <Modal
        title={t('批量切换用户分组')}
        visible={showBatchGroupModal}
        onCancel={() => {
          setShowBatchGroupModal(false);
          setTargetGroup('');
        }}
        onOk={handleConfirmBatchGroup}
      >
        <Space vertical align='start' spacing='medium' style={{ width: '100%' }}>
          <div>
            <Text>{t('已选择 {{count}} 个用户', { count: selectedUsers.length })}</Text>
          </div>
          <div className='flex flex-wrap gap-2'>
            {selectedUsers.slice(0, 8).map((user) => (
              <Tag key={user.id} color='blue'>
                {user.username}
              </Tag>
            ))}
            {selectedUsers.length > 8 && (
              <Tag color='grey'>
                {t('还有 {{count}} 个', { count: selectedUsers.length - 8 })}
              </Tag>
            )}
          </div>
          <Select
            filter
            allowCreate
            placeholder={t('请选择或输入目标分组')}
            optionList={groupOptions}
            value={targetGroup || undefined}
            onChange={(value) => setTargetGroup(value || '')}
            style={{ width: '100%' }}
          />
        </Space>
      </Modal>

      <Modal
        title={t('按分组整批切换用户')}
        visible={showMigrateGroupModal}
        onCancel={() => {
          setShowMigrateGroupModal(false);
          setSourceGroup('');
          setMigrateTargetGroup('');
        }}
        onOk={handleConfirmMigrateGroup}
      >
        <Space vertical align='start' spacing='medium' style={{ width: '100%' }}>
          <Text type='secondary'>
            {t('将所有当前属于某个分组的用户，一次性切换到另一个分组')}
          </Text>
          <div style={{ width: '100%' }}>
            <Text className='block mb-1'>{t('来源分组')}</Text>
            <Select
              filter
              allowCreate
              placeholder={t('请选择或输入来源分组')}
              optionList={groupOptions}
              value={sourceGroup || undefined}
              onChange={(value) => setSourceGroup(value || '')}
              style={{ width: '100%' }}
            />
          </div>
          <div style={{ width: '100%' }}>
            <Text className='block mb-1'>{t('目标分组')}</Text>
            <Select
              filter
              allowCreate
              placeholder={t('请选择或输入目标分组')}
              optionList={groupOptions}
              value={migrateTargetGroup || undefined}
              onChange={(value) => setMigrateTargetGroup(value || '')}
              style={{ width: '100%' }}
            />
          </div>
        </Space>
      </Modal>
    </>
  );
};

export default UsersActions;
