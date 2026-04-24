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

import { useState, useEffect } from 'react';
import { useTranslation } from 'react-i18next';
import { API, showError, showSuccess } from '../../helpers';
import { ITEMS_PER_PAGE } from '../../constants';
import { useTableCompactMode } from '../common/useTableCompactMode';

export const useUsersData = () => {
  const { t } = useTranslation();
  const [compactMode, setCompactMode] = useTableCompactMode('users');

  // State management
  const [users, setUsers] = useState([]);
  const [loading, setLoading] = useState(true);
  const [activePage, setActivePage] = useState(1);
  const [pageSize, setPageSize] = useState(ITEMS_PER_PAGE);
  const [searching, setSearching] = useState(false);
  const [groupOptions, setGroupOptions] = useState([]);
  const [userCount, setUserCount] = useState(0);
  const [selectedUsers, setSelectedUsers] = useState([]);

  // Modal states
  const [showAddUser, setShowAddUser] = useState(false);
  const [showEditUser, setShowEditUser] = useState(false);
  const [editingUser, setEditingUser] = useState({
    id: undefined,
  });

  // Form initial values
  const formInitValues = {
    searchKeyword: '',
    searchGroup: '',
    searchSort: '',
  };

  const rowSelection = {
    getCheckboxProps: (record) => ({
      name: record.username,
    }),
    selectedRowKeys: selectedUsers.map((user) => user.id),
    onChange: (selectedRowKeys, selectedRows) => {
      setSelectedUsers(selectedRows);
    },
  };

  // Form API reference
  const [formApi, setFormApi] = useState(null);

  // Get form values helper function
  const getFormValues = () => {
    const formValues = formApi ? formApi.getValues() : {};
    return {
      searchKeyword: formValues.searchKeyword || '',
      searchGroup: formValues.searchGroup || '',
      searchSort: formValues.searchSort || '',
    };
  };

  const buildUserListParams = (page, size, sortBy = '') => {
    const params = new URLSearchParams({
      p: String(page),
      page_size: String(size),
    });
    if (sortBy) {
      params.set('sort', sortBy);
    }
    return params.toString();
  };

  // Set user format with key field
  const setUserFormat = (users) => {
    for (let i = 0; i < users.length; i++) {
      users[i].key = users[i].id;
    }
    setUsers(users);
  };

  // Load users data
  const loadUsers = async (startIdx, pageSize, sortBy = null) => {
    setLoading(true);
    const finalSort = sortBy ?? getFormValues().searchSort;
    const res = await API.get(
      `/api/user/?${buildUserListParams(startIdx, pageSize, finalSort)}`,
    );
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setUserCount(data.total);
      setUserFormat(newPageData);
    } else {
      showError(message);
    }
    setLoading(false);
  };

  // Search users with keyword and group
  const searchUsers = async (
    startIdx,
    pageSize,
    searchKeyword = null,
    searchGroup = null,
    searchSort = null,
  ) => {
    // If no parameters passed, get values from form
    if (searchKeyword === null || searchGroup === null || searchSort === null) {
      const formValues = getFormValues();
      searchKeyword = formValues.searchKeyword;
      searchGroup = formValues.searchGroup;
      searchSort = formValues.searchSort;
    }

    if (searchKeyword === '' && searchGroup === '') {
      // If keyword is blank, load files instead
      await loadUsers(startIdx, pageSize, searchSort);
      return;
    }
    setSearching(true);
    const params = new URLSearchParams({
      keyword: searchKeyword,
      group: searchGroup,
      p: String(startIdx),
      page_size: String(pageSize),
    });
    if (searchSort) {
      params.set('sort', searchSort);
    }
    const res = await API.get(`/api/user/search?${params.toString()}`);
    const { success, message, data } = res.data;
    if (success) {
      const newPageData = data.items;
      setActivePage(data.page);
      setUserCount(data.total);
      setUserFormat(newPageData);
    } else {
      showError(message);
    }
    setSearching(false);
  };

  // Manage user operations (promote, demote, enable, disable, delete)
  const manageUser = async (userId, action, record) => {
    // Trigger loading state to force table re-render
    setLoading(true);

    const res = await API.post('/api/user/manage', {
      id: userId,
      action,
    });

    const { success, message } = res.data;
    if (success) {
      showSuccess(t('操作成功完成！'));
      const user = res.data.data;

      // Create a new array and new object to ensure React detects changes
      const newUsers = users.map((u) => {
        if (u.id === userId) {
          if (action === 'delete') {
            return { ...u, DeletedAt: new Date() };
          }
          return { ...u, status: user.status, role: user.role };
        }
        return u;
      });

      setUsers(newUsers);
    } else {
      showError(message);
    }

    setLoading(false);
  };

  const resetUserPasskey = async (user) => {
    if (!user) {
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/reset_passkey`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('Passkey 已重置'));
      } else {
        showError(message || t('操作失败，请重试'));
      }
    } catch (error) {
      showError(t('操作失败，请重试'));
    }
  };

  const resetUserTwoFA = async (user) => {
    if (!user) {
      return;
    }
    try {
      const res = await API.delete(`/api/user/${user.id}/2fa`);
      const { success, message } = res.data;
      if (success) {
        showSuccess(t('二步验证已重置'));
      } else {
        showError(message || t('操作失败，请重试'));
      }
    } catch (error) {
      showError(t('操作失败，请重试'));
    }
  };

  // Handle page change
  const handlePageChange = (page) => {
    setActivePage(page);
    const { searchKeyword, searchGroup, searchSort } = getFormValues();
    if (searchKeyword === '' && searchGroup === '') {
      loadUsers(page, pageSize, searchSort).then();
    } else {
      searchUsers(page, pageSize, searchKeyword, searchGroup, searchSort).then();
    }
  };

  // Handle page size change
  const handlePageSizeChange = async (size) => {
    localStorage.setItem('page-size', size + '');
    setPageSize(size);
    setActivePage(1);
    const { searchKeyword, searchGroup, searchSort } = getFormValues();
    const request =
      searchKeyword === '' && searchGroup === ''
        ? loadUsers(1, size, searchSort)
        : searchUsers(1, size, searchKeyword, searchGroup, searchSort);
    request.then().catch((reason) => {
      showError(reason);
    });
  };

  // Handle table row styling for disabled/deleted users
  const handleRow = (record, index) => {
    const rowStyle =
      record.DeletedAt !== null || record.status !== 1
        ? {
            style: {
              background: 'var(--semi-color-disabled-border)',
            },
          }
        : {};

    return {
      ...rowStyle,
      onClick: (event) => {
        if (
          event.target.closest(
            'button, .semi-button, a, input, textarea, .semi-checkbox, [role="button"]',
          )
        ) {
          return;
        }
        const nextSelectedUsers = selectedUsers.some(
          (user) => user.id === record.id,
        )
          ? selectedUsers.filter((user) => user.id !== record.id)
          : [...selectedUsers, record];
        setSelectedUsers(nextSelectedUsers);
      },
    };
  };

  // Refresh data
  const refresh = async (page = activePage) => {
    const { searchKeyword, searchGroup, searchSort } = getFormValues();
    if (searchKeyword === '' && searchGroup === '') {
      await loadUsers(page, pageSize, searchSort);
    } else {
      await searchUsers(page, pageSize, searchKeyword, searchGroup, searchSort);
    }
  };

  // Fetch groups data
  const fetchGroups = async () => {
    try {
      let res = await API.get(`/api/group/`);
      if (res === undefined) {
        return;
      }
      setGroupOptions(
        res.data.data.map((group) => ({
          label: group,
          value: group,
        })),
      );
    } catch (error) {
      showError(error.message);
    }
  };

  // Modal control functions
  const closeAddUser = () => {
    setShowAddUser(false);
  };

  const closeEditUser = () => {
    setShowEditUser(false);
    setEditingUser({
      id: undefined,
    });
  };

  // Initialize data on component mount
  useEffect(() => {
    loadUsers(0, pageSize)
      .then()
      .catch((reason) => {
        showError(reason);
      });
    fetchGroups().then();
  }, []);

  const batchSwitchUserGroup = async (targetGroup) => {
    const finalGroup = (targetGroup || '').trim();
    if (selectedUsers.length === 0) {
      showError(t('请至少选择一个用户'));
      return false;
    }
    if (finalGroup === '') {
      showError(t('请选择目标分组'));
      return false;
    }

    const res = await API.post('/api/user/batch/group', {
      user_ids: selectedUsers.map((user) => user.id),
      target_group: finalGroup,
    });
    const { success, message, data } = res.data;
    if (!success) {
      showError(message);
      return false;
    }

    showSuccess(
      t('已批量切换 {{count}} 个用户到分组 {{group}}', {
        count: data?.user_count || selectedUsers.length,
        group: finalGroup,
      }),
    );
    setSelectedUsers([]);
    await refresh();
    return true;
  };

  const migrateUserGroup = async (sourceGroup, targetGroup) => {
    const finalSourceGroup = (sourceGroup || '').trim();
    const finalTargetGroup = (targetGroup || '').trim();
    if (finalSourceGroup === '' || finalTargetGroup === '') {
      showError(t('请选择来源分组和目标分组'));
      return false;
    }
    if (finalSourceGroup === finalTargetGroup) {
      showError(t('来源分组和目标分组不能相同'));
      return false;
    }

    const res = await API.post('/api/user/migrate_group', {
      source_group: finalSourceGroup,
      target_group: finalTargetGroup,
    });
    const { success, message, data } = res.data;
    if (!success) {
      showError(message);
      return false;
    }

    showSuccess(
      t('已将分组 {{source}} 的 {{count}} 个用户切换到 {{target}}', {
        source: finalSourceGroup,
        count: data?.user_count || 0,
        target: finalTargetGroup,
      }),
    );
    setSelectedUsers([]);
    await refresh(0);
    return true;
  };

  return {
    // Data state
    users,
    loading,
    activePage,
    pageSize,
    userCount,
    searching,
    groupOptions,
    selectedUsers,
    rowSelection,
    setSelectedUsers,

    // Modal state
    showAddUser,
    showEditUser,
    editingUser,
    setShowAddUser,
    setShowEditUser,
    setEditingUser,

    // Form state
    formInitValues,
    formApi,
    setFormApi,

    // UI state
    compactMode,
    setCompactMode,

    // Actions
    loadUsers,
    searchUsers,
    manageUser,
    resetUserPasskey,
    resetUserTwoFA,
    handlePageChange,
    handlePageSizeChange,
    handleRow,
    refresh,
    batchSwitchUserGroup,
    migrateUserGroup,
    closeAddUser,
    closeEditUser,
    getFormValues,

    // Translation
    t,
  };
};
