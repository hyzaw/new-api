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
import { Card, Typography } from '@douyinfe/semi-ui';

const { Text, Title } = Typography;

export function AuthShell({
  brandLogo,
  systemName,
  panelEyebrow,
  panelTitle,
  panelDescription,
  highlights,
  children,
  turnstile,
}) {
  return (
    <div className='auth-business-shell min-h-screen overflow-hidden px-4 py-10 sm:px-6 lg:px-8'>
      <div className='auth-business-grid' />
      <div className='relative mx-auto mt-[60px] grid w-full max-w-6xl items-center gap-8 lg:grid-cols-[minmax(0,1fr)_440px] lg:gap-12'>
        <div className='hidden lg:flex lg:flex-col lg:gap-8'>
          <div className='auth-business-badge'>
            <span className='home-business-badge-dot' />
            <span>{panelEyebrow}</span>
          </div>
          <div className='max-w-2xl'>
            <Text className='!text-xs !font-semibold uppercase tracking-[0.28em] !text-semi-color-text-2'>
              {systemName}
            </Text>
            <Title heading={1} className='!mb-0 !mt-4 auth-business-title'>
              {panelTitle}
            </Title>
            <Text className='auth-business-description !mt-5'>
              {panelDescription}
            </Text>
          </div>
          <div className='grid gap-4'>
            {highlights.map((item) => (
              <div className='auth-business-feature' key={item.title}>
                <Text className='auth-business-feature-title'>{item.title}</Text>
                <Text className='auth-business-feature-text'>
                  {item.description}
                </Text>
              </div>
            ))}
          </div>
        </div>
        <div className='relative z-[1] mx-auto flex w-full max-w-md flex-col gap-5'>
          {children}
          {turnstile}
        </div>
      </div>
    </div>
  );
}

export function AuthCard({
  brandLogo,
  systemName,
  title,
  subtitle,
  children,
}) {
  return (
    <Card className='auth-business-card border-0 !rounded-[28px] overflow-hidden'>
      <div className='px-6 pt-7 md:px-8 md:pt-8'>
        <div className='flex items-center gap-3'>
          <img
            src={brandLogo}
            alt='Logo'
            className='h-12 w-12 rounded-2xl object-cover shadow-sm'
          />
          <div className='min-w-0'>
            <Text className='!block !text-xs !font-semibold uppercase tracking-[0.24em] !text-semi-color-text-2'>
              {systemName}
            </Text>
            <Title heading={3} className='!mb-0 !mt-2 auth-business-card-title'>
              {title}
            </Title>
          </div>
        </div>
        {subtitle ? (
          <Text className='auth-business-card-subtitle !mt-4 !block'>
            {subtitle}
          </Text>
        ) : null}
      </div>
      <div className='px-6 pb-7 pt-6 md:px-8 md:pb-8'>{children}</div>
    </Card>
  );
}
