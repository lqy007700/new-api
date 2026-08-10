/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.
*/
import { ChevronLeft, ChevronRight, Gift } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Skeleton } from '@/components/ui/skeleton'
import { formatQuota, formatTimestamp } from '@/lib/format'

import { useRedemptionHistory } from '../../hooks/use-redemption-history'

interface RedemptionHistoryDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function RedemptionHistoryDialog({
  open,
  onOpenChange,
}: RedemptionHistoryDialogProps) {
  const { t } = useTranslation()
  const {
    records,
    total,
    page,
    pageSize,
    loading,
    handlePageChange,
    handlePageSizeChange,
  } = useRedemptionHistory()

  const totalPages = Math.ceil(total / pageSize)

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Redemption History')}
      description={t('View your redemption records and credited quota')}
      contentClassName='flex max-h-[calc(100dvh-2rem)] flex-col max-sm:w-screen max-sm:max-w-none max-sm:rounded-none max-sm:p-4 sm:max-w-4xl'
      contentHeight='auto'
      bodyClassName='space-y-3'
    >
      <div className='min-h-0 space-y-3'>
        <div className='flex justify-end'>
          <Select
            items={[
              { value: '10', label: t('10 / page') },
              { value: '20', label: t('20 / page') },
              { value: '50', label: t('50 / page') },
              { value: '100', label: t('100 / page') },
            ]}
            value={pageSize.toString()}
            onValueChange={(value) =>
              value !== null && handlePageSizeChange(Number.parseInt(value))
            }
          >
            <SelectTrigger className='h-9 w-[92px] sm:w-32'>
              <SelectValue />
            </SelectTrigger>
            <SelectContent alignItemWithTrigger={false}>
              <SelectGroup>
                <SelectItem value='10'>{t('10 / page')}</SelectItem>
                <SelectItem value='20'>{t('20 / page')}</SelectItem>
                <SelectItem value='50'>{t('50 / page')}</SelectItem>
                <SelectItem value='100'>{t('100 / page')}</SelectItem>
              </SelectGroup>
            </SelectContent>
          </Select>
        </div>

        <div className='max-h-[min(54vh,520px)] overflow-y-auto pr-1'>
          {loading ? (
            <RedemptionHistorySkeleton />
          ) : (
            <RedemptionHistoryRecords records={records} t={t} />
          )}
        </div>

        {!loading && records.length > 0 && (
          <div className='flex flex-col items-center gap-3 border-t pt-4 sm:flex-row sm:items-center sm:justify-between'>
            <div className='text-muted-foreground text-xs sm:text-sm'>
              {t('Showing')} {(page - 1) * pageSize + 1}-
              {Math.min(page * pageSize, total)} {t('of')} {total}
            </div>
            <div className='flex items-center gap-2'>
              <Button
                variant='outline'
                size='sm'
                onClick={() => handlePageChange(page - 1)}
                disabled={page <= 1}
                className='h-8 w-8 p-0'
              >
                <ChevronLeft className='h-4 w-4' />
              </Button>
              <div className='text-muted-foreground flex items-center gap-1 text-sm'>
                <span className='font-medium'>{page}</span>
                <span>/</span>
                <span>{totalPages}</span>
              </div>
              <Button
                variant='outline'
                size='sm'
                onClick={() => handlePageChange(page + 1)}
                disabled={page >= totalPages}
                className='h-8 w-8 p-0'
              >
                <ChevronRight className='h-4 w-4' />
              </Button>
            </div>
          </div>
        )}
      </div>
    </Dialog>
  )
}

function RedemptionHistorySkeleton() {
  return (
    <div className='space-y-3'>
      {Array.from(
        { length: 5 },
        (_, index) => `redemption-skeleton-${index}`
      ).map((key) => (
        <div key={key} className='rounded-lg border p-3 sm:p-4'>
          <Skeleton className='h-4 w-48' />
          <div className='mt-3 grid grid-cols-2 gap-3 sm:grid-cols-3 sm:gap-4'>
            <Skeleton className='h-3 w-full' />
            <Skeleton className='h-3 w-full' />
            <Skeleton className='h-3 w-full' />
          </div>
        </div>
      ))}
    </div>
  )
}

function RedemptionHistoryRecords({
  records,
  t,
}: {
  records: Array<{
    id: number
    name: string
    quota: number
    redeemed_time: number
  }>
  t: (key: string) => string
}) {
  if (records.length === 0) {
    return (
      <div className='text-muted-foreground flex min-h-40 flex-col items-center justify-center py-10 text-center'>
        <Gift className='mb-3 h-8 w-8 opacity-50' />
        <p className='text-sm font-medium'>
          {t('No redemption records found')}
        </p>
        <p className='mt-1 text-xs'>
          {t('Your redemption history will appear here')}
        </p>
      </div>
    )
  }

  return (
    <div className='space-y-3'>
      {records.map((record) => (
        <div key={record.id} className='rounded-lg border p-3 sm:p-4'>
          <div className='flex items-start justify-between gap-2'>
            <div className='flex items-center gap-2'>
              <Gift className='h-4 w-4 text-amber-500' />
              <div className='text-muted-foreground text-xs'>
                {formatTimestamp(record.redeemed_time)}
              </div>
            </div>
            <div className='text-sm font-semibold text-emerald-600'>
              +{formatQuota(record.quota)}
            </div>
          </div>

          <div className='mt-3 grid grid-cols-2 gap-3 sm:mt-4 sm:grid-cols-3 sm:gap-4'>
            <div className='space-y-1'>
              <Label className='text-muted-foreground text-xs'>
                {t('Amount')}
              </Label>
              <div className='text-sm font-semibold'>
                {formatQuota(record.quota)}
              </div>
            </div>
            <div className='space-y-1'>
              <Label className='text-muted-foreground text-xs'>
                {t('Redemption Details')}
              </Label>
              <div className='truncate text-sm font-medium'>
                {record.name || t('Redemption Code')}
              </div>
            </div>
            <div className='space-y-1'>
              <Label className='text-muted-foreground text-xs'>
                {t('Record ID')}
              </Label>
              <div className='text-sm font-medium'>#{record.id}</div>
            </div>
          </div>
        </div>
      ))}
    </div>
  )
}
