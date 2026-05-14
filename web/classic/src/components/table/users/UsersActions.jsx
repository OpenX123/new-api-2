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

import React, { useState } from 'react';
import { Button } from '@douyinfe/semi-ui';
import { showError } from '../../../helpers';
import BatchDeleteUserModal from './modals/BatchDeleteUserModal';

const UsersActions = ({ setShowAddUser, selectedKeys, batchDeleteUsers, t }) => {
  const [showBatchDeleteModal, setShowBatchDeleteModal] = useState(false);

  const handleAddUser = () => {
    setShowAddUser(true);
  };

  const handleBatchDelete = () => {
    if (!selectedKeys || selectedKeys.length === 0) {
      showError(t('请至少选择一个用户！'));
      return;
    }
    setShowBatchDeleteModal(true);
  };

  const handleBatchDeleteConfirm = async () => {
    setShowBatchDeleteModal(false);
    await batchDeleteUsers();
  };

  return (
    <>
      <div className='flex gap-2 w-full md:w-auto order-2 md:order-1'>
        <Button className='w-full md:w-auto' onClick={handleAddUser} size='small'>
          {t('添加用户')}
        </Button>
        <Button
          className='w-full md:w-auto'
          type='danger'
          onClick={handleBatchDelete}
          size='small'
        >
          {t('批量删除')}
        </Button>
      </div>

      <BatchDeleteUserModal
        visible={showBatchDeleteModal}
        onCancel={() => setShowBatchDeleteModal(false)}
        onConfirm={handleBatchDeleteConfirm}
        selectedKeys={selectedKeys || []}
        t={t}
      />
    </>
  );
};

export default UsersActions;
