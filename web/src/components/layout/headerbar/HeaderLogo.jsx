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
import { Link } from 'react-router-dom';
import { Tag } from '@douyinfe/semi-ui';
import SkeletonWrapper from '../components/SkeletonWrapper';

const HeaderLogo = ({
  isMobile,
  isConsoleRoute,
  isLoading,
  isSelfUseMode,
  isDemoSiteMode,
  t,
}) => {
  if (isMobile && isConsoleRoute) {
    return null;
  }

  return (
    <Link to='/' className='group flex items-center gap-2'>
      <div className='hidden md:flex items-center gap-2'>
        <SkeletonWrapper loading={isLoading} type='title' width={168} height={36}>
          <img
            src='/logo-svg.svg'
            alt='Boxying'
            className='h-9 w-auto transition-transform duration-200 group-hover:scale-[1.03]'
          />
        </SkeletonWrapper>
        {(isSelfUseMode || isDemoSiteMode) && !isLoading && (
          <Tag
            color={isSelfUseMode ? 'purple' : 'blue'}
            className='text-xs px-1.5 py-0.5 rounded whitespace-nowrap shadow-sm'
            size='small'
            shape='circle'
          >
            {isSelfUseMode ? t('自用模式') : t('演示站点')}
          </Tag>
        )}
      </div>
      <div className='flex md:hidden items-center'>
        <img
          src='/favicon.svg'
          alt='Boxying'
          className='w-9 h-9 transition-transform duration-200 group-hover:scale-105'
        />
      </div>
    </Link>
  );
};

export default HeaderLogo;
