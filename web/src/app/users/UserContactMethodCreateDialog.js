import React, { useState } from 'react'
import p from 'prop-types'

import gql from 'graphql-tag'
import { fieldErrors, nonFieldErrors } from '../util/errutil'

import FormDialog from '../dialogs/FormDialog'
import UserContactMethodForm from './UserContactMethodForm'
import { useMutation } from '@apollo/react-hooks'

const createMutation = gql`
  mutation($input: CreateUserContactMethodInput!) {
    createUserContactMethod(input: $input) {
      id
    }
  }
`

export default function UserContactMethodCreateDialog(props) {
  // values for contact method form
  const [cmValue, setCmValue] = useState({
    name: '',
    type: 'SMS',
    value: '',
  })

  const [createCM, createCMStatus] = useMutation(createMutation, {
    refetchQueries: ['nrList', 'cmList'],
    awaitRefetchQueries: true,
    onCompleted: result => {
      props.onClose({ contactMethodID: result.createUserContactMethod.id })
    },
    variables: {
      input: {
        ...cmValue,
        userID: props.userID,
        newUserNotificationRule: {
          delayMinutes: 0,
        },
      },
    },
  })

  const { loading, error } = createCMStatus
  const fieldErrs = fieldErrors(error)
  const form = (
    <UserContactMethodForm
      disabled={loading}
      errors={fieldErrs}
      onChange={cmValue => setCmValue(cmValue)}
      value={cmValue}
    />
  )
  return (
    <FormDialog
      data-cy='create-form'
      title='Create New Contact Method'
      loading={loading}
      errors={nonFieldErrors(error)}
      onClose={props.onClose}
      // wrapped to prevent event from passing into createCM
      onSubmit={() => createCM()}
      form={form}
    />
  )
}

UserContactMethodCreateDialog.propTypes = {
  userID: p.string.isRequired,
  onClose: p.func,
}
